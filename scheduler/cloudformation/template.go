package cloudformation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/troposphere"
	"github.com/remind101/empire/twelvefactor"
)

var (
	Ref    = troposphere.Ref
	GetAtt = troposphere.GetAtt
	Equals = troposphere.Equals
	Join   = troposphere.Join
)

// Load balancer types
const (
	classicLoadBalancer     = "elb"
	applicationLoadBalancer = "alb"
)

// Returns the type of load balancer that should be used (ELB/ALB).
func loadBalancerType(app *twelvefactor.Manifest, process *twelvefactor.Process) string {
	check := []string{
		"EMPIRE_X_LOAD_BALANCER_TYPE",
		"LOAD_BALANCER_TYPE", // For backwards compatibility.
	}
	env := twelvefactor.Env(app, process)

	for _, n := range check {
		if v, ok := env[n]; ok {
			return v
		}
	}

	// Default when not set.
	return classicLoadBalancer
}

// Returns the name of the CloudFormation resource that should be used to create
// custom task definitions.
func taskDefinitionResourceType(app *twelvefactor.Manifest) string {
	check := []string{
		"EMPIRE_X_TASK_DEFINITION_TYPE",
		"ECS_TASK_DEFINITION", // For backwards compatibility.
	}

	for _, n := range check {
		if v, ok := app.Env[n]; ok {
			if v == "custom" {
				return "Custom::ECSTaskDefinition"
			}
		}
	}

	// Default when not set.
	return "AWS::ECS::TaskDefinition"
}

func taskRoleArn(app *twelvefactor.Manifest) *string {
	check := []string{
		"EMPIRE_X_TASK_ROLE_ARN",
		"TASK_ROLE_ARN", // For backwards compatibility.
	}

	for _, n := range check {
		if v, ok := app.Env[n]; ok {
			return &v
		}
	}

	return nil
}

const (
	schemeInternal = "internal"
	schemeExternal = "internet-facing"

	defaultConnectionDrainingTimeout int64 = 30
	defaultCNAMETTL                        = 60

	runTaskFunction = "RunTaskFunction"

	appEnvironment = "AppEnvironment"

	restartLabel = "cloudformation.restart-key"
)

// This implements the Template interface to create a suitable CloudFormation
// template for an Empire app.
type EmpireTemplate struct {
	// By default, the JSON will not have any whitespace or newlines, which
	// helps prevent templates from going over the maximum size limit. If
	// you care about readability, you can set this to true.
	NoCompress bool

	// The ECS cluster to run the services in.
	Cluster string

	// The VPC to create ALB target groups within. Should be the same VPC
	// that ECS services will run within.
	VpcId string

	// The hosted zone to add CNAME's to.
	HostedZone *route53.HostedZone

	// The ID of the security group to assign to internal load balancers.
	InternalSecurityGroupID string

	// The ID of the security group to assign to external load balancers.
	ExternalSecurityGroupID string

	// The Subnet IDs to assign when creating internal load balancers.
	InternalSubnetIDs []string

	// The Subnet IDs to assign when creating external load balancers.
	ExternalSubnetIDs []string

	// The name or ARN of the IAM role to allow ECS, CloudWatch Events, and Lambda
	// to assume.
	ServiceRole string

	// The ARN of the SNS topic to provision instance ports.
	CustomResourcesTopic string

	LogConfiguration *ecs.LogConfiguration

	// Any extra outputs to attach to the template.
	ExtraOutputs map[string]troposphere.Output
}

// Validate checks that all of the expected values are provided.
func (t *EmpireTemplate) Validate() error {
	r := func(n string) error {
		return errors.New(fmt.Sprintf("%s is required", n))
	}

	if t.VpcId == "" {
		return r("VpcId")
	}
	if t.Cluster == "" {
		return r("Cluster")
	}
	if t.ServiceRole == "" {
		return r("ServiceRole")
	}
	if t.HostedZone == nil {
		return r("HostedZone")
	}
	if t.InternalSecurityGroupID == "" {
		return r("InternalSecurityGroupID")
	}
	if t.ExternalSecurityGroupID == "" {
		return r("ExternalSecurityGroupID")
	}
	if len(t.InternalSubnetIDs) == 0 {
		return r("InternalSubnetIDs")
	}
	if len(t.ExternalSubnetIDs) == 0 {
		return r("ExternalSubnetIDs")
	}
	if t.CustomResourcesTopic == "" {
		return r("CustomResourcesTopic")
	}

	return nil
}

// Execute builds the template, and writes it to w.
func (t *EmpireTemplate) Execute(w io.Writer, data interface{}) error {
	v, err := t.Build(data.(*TemplateData))
	if err != nil {
		return err
	}

	if t.NoCompress {
		raw, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			return err
		}

		_, err = io.Copy(w, bytes.NewReader(raw))
		return err
	}

	return json.NewEncoder(w).Encode(v)
}

// Build builds a Go representation of a CloudFormation template for the app.
func (t *EmpireTemplate) Build(data *TemplateData) (*troposphere.Template, error) {
	app := data.Manifest

	tmpl := troposphere.NewTemplate()

	tmpl.Parameters["DNS"] = troposphere.Parameter{
		Type:        "String",
		Description: "When set to `true`, CNAME's will be altered",
		Default:     "true",
	}
	tmpl.Parameters[restartParameter] = troposphere.Parameter{
		Type:        "String",
		Description: "Key used to trigger a restart of an app",
		Default:     "default",
	}
	tmpl.Conditions["DNSCondition"] = Equals(Ref("DNS"), "true")

	for k, v := range t.ExtraOutputs {
		tmpl.Outputs[k] = v
	}

	tmpl.Outputs["Release"] = troposphere.Output{Value: app.Release}

	serviceMappings := []interface{}{}
	deploymentMappings := []interface{}{}
	scheduledProcesses := map[string]string{}

	if taskDefinitionResourceType(app) == "Custom::ECSTaskDefinition" {
		tmpl.Resources[appEnvironment] = troposphere.Resource{
			Type: "Custom::ECSEnvironment",
			Properties: map[string]interface{}{
				"ServiceToken": t.CustomResourcesTopic,
				"Environment":  sortedEnvironment(app.Env),
			},
		}
	}

	for _, p := range app.Processes {
		if p.Env == nil {
			p.Env = make(map[string]string)
		}

		tmpl.Parameters[scaleParameter(p.Type)] = troposphere.Parameter{
			Type: "String",
		}

		switch {
		case p.Schedule != nil:
			taskDefinition := t.addScheduledTask(tmpl, app, p)
			scheduledProcesses[p.Type] = taskDefinition.Name
		default:
			service, err := t.addService(tmpl, app, p, data.StackTags)
			if err != nil {
				return tmpl, err
			}
			serviceMappings = append(serviceMappings, Join("=", p.Type, Ref(service)))
			deploymentMappings = append(deploymentMappings, Join("=", p.Type, GetAtt(service, "DeploymentId")))
		}
	}

	if len(scheduledProcesses) > 0 {
		// LambdaFunction that will be used to trigger a RunTask.
		tmpl.Resources[runTaskFunction] = runTaskResource(t.serviceRoleArn())
	}

	tmpl.Outputs[servicesOutput] = troposphere.Output{Value: Join(",", serviceMappings...)}
	tmpl.Outputs[deploymentsOutput] = troposphere.Output{Value: Join(",", deploymentMappings...)}

	return tmpl, nil
}

func (t *EmpireTemplate) addTaskDefinition(tmpl *troposphere.Template, app *twelvefactor.Manifest, p *twelvefactor.Process) (troposphere.NamedResource, *ContainerDefinitionProperties) {
	key := processResourceName(p.Type)
	// The task definition that will be used to run the ECS task.
	taskDefinition := troposphere.NamedResource{
		Name: fmt.Sprintf("%sTaskDefinition", key),
	}

	cd := t.ContainerDefinition(app, p)
	containerDefinition := cloudformationContainerDefinition(cd)

	// If provided in the app environment, this role will be used when
	// running tasks.
	taskRole := toInterface(taskRoleArn(app))

	var taskDefinitionProperties interface{}
	taskDefinitionType := taskDefinitionResourceType(app)
	if taskDefinitionType == "Custom::ECSTaskDefinition" {
		taskDefinition.Name = fmt.Sprintf("%sTD", key)

		processEnvironment := fmt.Sprintf("%sEnvironment", key)
		tmpl.Resources[processEnvironment] = troposphere.Resource{
			Type: "Custom::ECSEnvironment",
			Properties: map[string]interface{}{
				"ServiceToken": t.CustomResourcesTopic,
				"Environment":  sortedEnvironment(p.Env),
			},
		}

		containerDefinition.Environment = []interface{}{
			Ref(appEnvironment),
			Ref(processEnvironment),
		}
		taskDefinitionProperties = &CustomTaskDefinitionProperties{
			Volumes:      []interface{}{},
			ServiceToken: t.CustomResourcesTopic,
			Family:       fmt.Sprintf("%s-%s", app.Name, p.Type),
			ContainerDefinitions: []*ContainerDefinitionProperties{
				containerDefinition,
			},
			TaskRoleArn: taskRole,
		}
	} else {
		containerDefinition.Environment = cd.Environment
		taskDefinitionProperties = &TaskDefinitionProperties{
			Volumes: []interface{}{},
			ContainerDefinitions: []*ContainerDefinitionProperties{
				containerDefinition,
			},
			TaskRoleArn: taskRole,
		}
	}

	taskDefinition.Resource = troposphere.Resource{
		Type:       taskDefinitionType,
		Properties: taskDefinitionProperties,
	}
	tmpl.AddResource(taskDefinition)

	return taskDefinition, containerDefinition
}

func (t *EmpireTemplate) addScheduledTask(tmpl *troposphere.Template, app *twelvefactor.Manifest, p *twelvefactor.Process) troposphere.NamedResource {
	key := processResourceName(p.Type)

	taskDefinition, _ := t.addTaskDefinition(tmpl, app, p)

	state := "DISABLED"
	if p.Quantity > 0 {
		state = "ENABLED"
	}
	schedule := fmt.Sprintf("%sTrigger", key)
	tmpl.Resources[schedule] = troposphere.Resource{
		Type: "AWS::Events::Rule",
		Properties: map[string]interface{}{
			"Description":        fmt.Sprintf("Rule to periodically trigger the `%s` scheduled task", p.Type),
			"ScheduleExpression": scheduleExpression(p.Schedule),
			"RoleArn":            t.serviceRoleArn(),
			"State":              state,
			"Targets": []interface{}{
				map[string]interface{}{
					"Arn":   GetAtt(runTaskFunction, "Arn"),
					"Id":    "f",
					"Input": Join("", `{"taskDefinition":"`, Ref(taskDefinition), `","count":`, Ref(scaleParameter(p.Type)), `,"cluster":"`, t.Cluster, `","startedBy": "`, app.AppID, `"}`),
				},
			},
		},
	}

	// Allow CloudWatch events to invoke the RunTask function.
	lambdaPermission := fmt.Sprintf("%sTriggerPermission", key)
	tmpl.Resources[lambdaPermission] = troposphere.Resource{
		Type: "AWS::Lambda::Permission",
		Properties: map[string]interface{}{
			"FunctionName": GetAtt(runTaskFunction, "Arn"),
			"SourceArn":    GetAtt(schedule, "Arn"),
			"Action":       "lambda:InvokeFunction",
			"Principal":    "events.amazonaws.com",
		},
	}

	return taskDefinition
}

func (t *EmpireTemplate) addService(tmpl *troposphere.Template, app *twelvefactor.Manifest, p *twelvefactor.Process, stackTags []*cloudformation.Tag) (serviceName string, err error) {
	key := processResourceName(p.Type)

	// Process specific tags to apply to resources.
	tags := tagsFromLabels(p.Labels)

	// The standard AWS::ECS::Service resource's default behavior is to wait
	// for services to stabilize when you update them. While this is a
	// sensible default for CloudFormation, the overall behavior when
	// applied to Empire is not a great experience, because updates will
	// lock up the stack.
	//
	// Setting this option makes the stack use a Custom::ECSService
	// resources intead, which does not wait for the service to stabilize
	// after updating.
	ecsServiceType := "Custom::ECSService"

	var portMappings []*PortMappingProperties

	var serviceDependencies []string
	loadBalancers := []map[string]interface{}{}
	if p.Exposure != nil {
		scheme := schemeInternal
		sg := t.InternalSecurityGroupID
		subnets := t.InternalSubnetIDs
		targetGroupPrefix := "Internal"

		if p.Exposure.External {
			scheme = schemeExternal
			sg = t.ExternalSecurityGroupID
			subnets = t.ExternalSubnetIDs
			targetGroupPrefix = "External"
		}

		loadBalancerType := loadBalancerType(app, p)

		var (
			loadBalancer          troposphere.NamedResource
			canonicalHostedZoneId interface{}
		)

		switch loadBalancerType {
		case applicationLoadBalancer:
			loadBalancer = troposphere.NamedResource{
				Name: fmt.Sprintf("%sApplicationLoadBalancer", key),
				Resource: troposphere.Resource{
					Type: "AWS::ElasticLoadBalancingV2::LoadBalancer",
					Properties: map[string]interface{}{
						"Scheme":         scheme,
						"SecurityGroups": []string{sg},
						"Subnets":        subnets,
						"Tags":           append(stackTags, tags...),
					},
				},
			}
			canonicalHostedZoneId = GetAtt(loadBalancer, "CanonicalHostedZoneID")

			tmpl.AddResource(loadBalancer)

			targetGroup := fmt.Sprintf("%s%sTargetGroup", key, targetGroupPrefix)
			log.Printf("using TargetGroup name %s\n", targetGroup)
			tmpl.Resources[targetGroup] = troposphere.Resource{
				Type: "AWS::ElasticLoadBalancingV2::TargetGroup",
				Properties: map[string]interface{}{
					"Port":     65535, // Not used. ECS sets a port override when registering targets.
					"Protocol": "HTTP",
					"VpcId":    t.VpcId,
					"Tags":     append(stackTags, tags...),
				},
			}

			// Add a port mapping for each unique container port.
			containerPorts := make(map[int]bool)
			for _, port := range p.Exposure.Ports {
				if ok := containerPorts[port.Container]; !ok {
					containerPorts[port.Container] = true
					portMappings = append(portMappings, &PortMappingProperties{
						ContainerPort: port.Container,
						HostPort:      0,
					})
				}
			}

			// Unlike ELB, ALB can only route to a single container
			// port, when dynamic ports are used. Thus, we have to
			// ensure that all of the defined ports map to the same
			// container port.
			//
			// ELB can route to multiple container ports, because a
			// listener can directly point to a container port,
			// through an instance port:
			//
			//	Listener Port => Instance Port => Container Port
			if len(containerPorts) > 1 {
				err = fmt.Errorf("AWS Application Load Balancers can only map listeners to a single container port. %d unique container ports were defined: [%s]", len(p.Exposure.Ports), fmtPorts(p.Exposure.Ports))
				return
			}

			// Add a listener for each port.
			for _, port := range p.Exposure.Ports {
				listener := troposphere.NamedResource{
					Name: fmt.Sprintf("%sPort%dListener", loadBalancer.Name, port.Host),
				}

				switch e := port.Protocol.(type) {
				case *twelvefactor.HTTP:
					listener.Resource = troposphere.Resource{
						Type: "AWS::ElasticLoadBalancingV2::Listener",
						Properties: map[string]interface{}{
							"LoadBalancerArn": Ref(loadBalancer),
							"Port":            port.Host,
							"Protocol":        "HTTP",
							"DefaultActions": []interface{}{
								map[string]interface{}{
									"TargetGroupArn": Ref(targetGroup),
									"Type":           "forward",
								},
							},
						},
					}
				case *twelvefactor.HTTPS:
					var cert interface{}
					if _, err := arn.Parse(e.Cert); err == nil {
						cert = e.Cert
					} else {
						cert = Join("", "arn:aws:iam::", Ref("AWS::AccountId"), ":server-certificate/", e.Cert)
					}

					listener.Resource = troposphere.Resource{
						Type: "AWS::ElasticLoadBalancingV2::Listener",
						Properties: map[string]interface{}{
							"Certificates": []interface{}{
								map[string]interface{}{
									"CertificateArn": cert,
								},
							},
							"LoadBalancerArn": Ref(loadBalancer),
							"Port":            port.Host,
							"Protocol":        "HTTPS",
							"DefaultActions": []interface{}{
								map[string]interface{}{
									"TargetGroupArn": Ref(targetGroup),
									"Type":           "forward",
								},
							},
						},
					}
				default:
					err = fmt.Errorf("%s listeners are not supported with AWS Application Load Balancing", e.Protocol())
					return
				}
				tmpl.AddResource(listener)
				serviceDependencies = append(serviceDependencies, listener.Name)
			}

			loadBalancers = append(loadBalancers, map[string]interface{}{
				"ContainerName":  p.Type,
				"ContainerPort":  p.Exposure.Ports[0].Container,
				"TargetGroupArn": Ref(targetGroup),
			})
		default:
			loadBalancer = troposphere.NamedResource{
				Name: fmt.Sprintf("%sLoadBalancer", key),
			}
			canonicalHostedZoneId = GetAtt(loadBalancer, "CanonicalHostedZoneNameID")

			listeners := []map[string]interface{}{}

			// Add a port mapping for each unique container port.
			instancePorts := make(map[int]troposphere.NamedResource)
			for _, port := range p.Exposure.Ports {
				if _, ok := instancePorts[port.Container]; !ok {
					instancePort := troposphere.NamedResource{
						Name: fmt.Sprintf("%s%dInstancePort", key, port.Container),
						Resource: troposphere.Resource{
							Type:    "Custom::InstancePort",
							Version: "1.0",
							Properties: map[string]interface{}{
								"ServiceToken": t.CustomResourcesTopic,
							},
						},
					}
					portMappings = append(portMappings, &PortMappingProperties{
						ContainerPort: port.Container,
						HostPort:      GetAtt(instancePort, "InstancePort"),
					})
					tmpl.AddResource(instancePort)
					instancePorts[port.Container] = instancePort
				}
			}

			for _, port := range p.Exposure.Ports {
				instancePort := instancePorts[port.Container]

				switch e := port.Protocol.(type) {
				case *twelvefactor.TCP:
					listeners = append(listeners, map[string]interface{}{
						"LoadBalancerPort": port.Host,
						"Protocol":         "tcp",
						"InstancePort":     GetAtt(instancePort, "InstancePort"),
						"InstanceProtocol": "tcp",
					})
				case *twelvefactor.SSL:
					var cert interface{}
					if _, err := arn.Parse(e.Cert); err == nil {
						cert = e.Cert
					} else {
						cert = Join("", "arn:aws:iam::", Ref("AWS::AccountId"), ":server-certificate/", e.Cert)
					}

					listeners = append(listeners, map[string]interface{}{
						"LoadBalancerPort": port.Host,
						"Protocol":         "ssl",
						"InstancePort":     GetAtt(instancePort, "InstancePort"),
						"SSLCertificateId": cert,
						"InstanceProtocol": "tcp",
					})
				case *twelvefactor.HTTP:
					listeners = append(listeners, map[string]interface{}{
						"LoadBalancerPort": port.Host,
						"Protocol":         "http",
						"InstancePort":     GetAtt(instancePort, "InstancePort"),
						"InstanceProtocol": "http",
					})
				case *twelvefactor.HTTPS:
					var cert interface{}
					if _, err := arn.Parse(e.Cert); err == nil {
						cert = e.Cert
					} else {
						cert = Join("", "arn:aws:iam::", Ref("AWS::AccountId"), ":server-certificate/", e.Cert)
					}

					listeners = append(listeners, map[string]interface{}{
						"LoadBalancerPort": port.Host,
						"Protocol":         "https",
						"InstancePort":     GetAtt(instancePort, "InstancePort"),
						"SSLCertificateId": cert,
						"InstanceProtocol": "http",
					})
				}
			}

			loadBalancer.Resource = troposphere.Resource{
				Type: "AWS::ElasticLoadBalancing::LoadBalancer",
				Properties: map[string]interface{}{
					"Scheme":         scheme,
					"SecurityGroups": []string{sg},
					"Subnets":        subnets,
					"Listeners":      listeners,
					"CrossZone":      true,
					"Tags":           tags,
					"ConnectionDrainingPolicy": map[string]interface{}{
						"Enabled": true,
						"Timeout": defaultConnectionDrainingTimeout,
					},
				},
			}
			tmpl.AddResource(loadBalancer)

			loadBalancers = append(loadBalancers, map[string]interface{}{
				"ContainerName":    p.Type,
				"ContainerPort":    p.Exposure.Ports[0].Container,
				"LoadBalancerName": Ref(loadBalancer),
			})
		}

		alias := troposphere.NamedResource{
			Name: fmt.Sprintf("%sAlias", key),
			Resource: troposphere.Resource{
				Type:      "AWS::Route53::RecordSet",
				Condition: "DNSCondition",
				Properties: map[string]interface{}{
					"HostedZoneId": *t.HostedZone.Id,
					"Name":         fmt.Sprintf("%s.%s.%s", p.Type, app.Name, *t.HostedZone.Name),
					"Type":         "A",
					"AliasTarget": map[string]interface{}{
						"DNSName":              GetAtt(loadBalancer, "DNSName"),
						"EvaluateTargetHealth": "true",
						"HostedZoneId":         canonicalHostedZoneId,
					},
				},
			},
		}
		tmpl.AddResource(alias)

		// DEPRECATED: This was used in the world where only the "web"
		// process could be exposed.
		if p.Type == "web" {
			tmpl.Resources["CNAME"] = troposphere.Resource{
				Type:      "AWS::Route53::RecordSet",
				Condition: "DNSCondition",
				Properties: map[string]interface{}{
					"HostedZoneId":    *t.HostedZone.Id,
					"Name":            fmt.Sprintf("%s.%s", app.Name, *t.HostedZone.Name),
					"Type":            "CNAME",
					"TTL":             defaultCNAMETTL,
					"ResourceRecords": []interface{}{GetAtt(loadBalancer, "DNSName")},
				},
			}
		}
	}

	taskDefinition, containerDefinition := t.addTaskDefinition(tmpl, app, p)

	containerDefinition.DockerLabels[restartLabel] = Ref(restartParameter)
	containerDefinition.PortMappings = portMappings

	serviceProperties := map[string]interface{}{
		"Cluster":        t.Cluster,
		"DesiredCount":   Ref(scaleParameter(p.Type)),
		"LoadBalancers":  loadBalancers,
		"TaskDefinition": Ref(taskDefinition),
		"ServiceName":    fmt.Sprintf("%s-%s", app.Name, p.Type),
		"ServiceToken":   t.CustomResourcesTopic,
	}
	if len(loadBalancers) > 0 {
		serviceProperties["Role"] = t.ServiceRole
	}
	service := troposphere.NamedResource{
		Name: fmt.Sprintf("%sService", key),
		Resource: troposphere.Resource{
			Type:       ecsServiceType,
			Properties: serviceProperties,
		},
	}
	if len(serviceDependencies) > 0 {
		service.Resource.DependsOn = serviceDependencies
	}
	tmpl.AddResource(service)
	return service.Name, nil
}

// If the ServiceRole option is not an ARN, it will return a CloudFormation
// expression that expands the ServiceRole to an ARN.
func (t *EmpireTemplate) serviceRoleArn() interface{} {
	if _, err := arn.Parse(t.ServiceRole); err == nil {
		return t.ServiceRole
	}
	return Join("", "arn:aws:iam::", Ref("AWS::AccountId"), ":role/", t.ServiceRole)
}

// ecsEnv implements the sort.Interface interface to sort the environment
// variables by key in alphabetical order.
type ecsEnv []*ecs.KeyValuePair

func (e ecsEnv) Len() int           { return len(e) }
func (e ecsEnv) Less(i, j int) bool { return *e[i].Name < *e[j].Name }
func (e ecsEnv) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

// ContainerDefinition generates an ECS ContainerDefinition for a process.
func (t *EmpireTemplate) ContainerDefinition(app *twelvefactor.Manifest, p *twelvefactor.Process) *ecs.ContainerDefinition {
	command := []*string{}
	for _, s := range p.Command {
		ss := s
		command = append(command, &ss)
	}

	labels := make(map[string]*string)
	for k, v := range twelvefactor.Labels(app, p) {
		labels[k] = aws.String(v)
	}

	ulimits := []*ecs.Ulimit{}
	if p.Nproc != 0 {
		ulimits = []*ecs.Ulimit{
			&ecs.Ulimit{
				Name:      aws.String("nproc"),
				SoftLimit: aws.Int64(int64(p.Nproc)),
				HardLimit: aws.Int64(int64(p.Nproc)),
			},
		}
	}

	return &ecs.ContainerDefinition{
		Name:             aws.String(p.Type),
		Cpu:              aws.Int64(int64(p.CPUShares)),
		Command:          command,
		Image:            aws.String(p.Image.String()),
		Essential:        aws.Bool(true),
		Memory:           aws.Int64(int64(p.Memory / bytesize.MB)),
		Environment:      sortedEnvironment(twelvefactor.Env(app, p)),
		LogConfiguration: t.LogConfiguration,
		DockerLabels:     labels,
		Ulimits:          ulimits,
	}
}

// HostedZone returns the HostedZone for the ZoneID.
func HostedZone(config client.ConfigProvider, hostedZoneID string) (*route53.HostedZone, error) {
	r := route53.New(config)
	zid := fixHostedZoneIDPrefix(hostedZoneID)
	out, err := r.GetHostedZone(&route53.GetHostedZoneInput{Id: zid})
	if err != nil {
		return nil, err
	}

	return out.HostedZone, nil
}

func fixHostedZoneIDPrefix(zoneID string) *string {
	prefix := "/hostedzone/"
	s := zoneID
	if ok := strings.HasPrefix(zoneID, prefix); !ok {
		s = strings.Join([]string{prefix, zoneID}, "")
	}
	return &s
}

// CloudFormation only allows alphanumeric resource names, so we
// have to normalize it.
var resourceRegex = regexp.MustCompile("[^a-zA-Z0-9]")

// processResourceName returns a string that can be used as a resource name in a
// CloudFormation stack for a process.
func processResourceName(process string) string {
	return resourceRegex.ReplaceAllString(process, "")
}

// scaleParameter returns the name of the parameter used to control the
// scale of a process.
func scaleParameter(process string) string {
	return fmt.Sprintf("%sScale", processResourceName(process))
}

// cloudformationContainerDefinition returns the CloudFormation representation
// of a ecs.ContainerDefinition.
func cloudformationContainerDefinition(cd *ecs.ContainerDefinition) *ContainerDefinitionProperties {
	labels := make(map[string]interface{})
	for k, v := range cd.DockerLabels {
		labels[k] = *v
	}

	c := &ContainerDefinitionProperties{
		Name:         *cd.Name,
		Command:      cd.Command,
		Cpu:          *cd.Cpu,
		Image:        *cd.Image,
		Essential:    *cd.Essential,
		Memory:       *cd.Memory,
		Environment:  cd.Environment,
		DockerLabels: labels,
		Ulimits:      cd.Ulimits,
	}
	if cd.LogConfiguration != nil {
		c.LogConfiguration = cd.LogConfiguration
	}
	return c
}

// sortedEnvironment takes a map[string]string and returns a sorted slice of
// ecs.KeyValuePair.
func sortedEnvironment(environment map[string]string) []*ecs.KeyValuePair {
	e := ecsEnv{}
	for k, v := range environment {
		e = append(e, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}
	sort.Sort(e)
	return e
}

func scheduleExpression(s twelvefactor.Schedule) string {
	switch v := s.(type) {
	case twelvefactor.CRONSchedule:
		return fmt.Sprintf("cron(%s)", v)
	case time.Duration:
		var units = "minute"
		var minutes = int64(v / time.Minute)
		if minutes > 1 {
			units += "s"
		}
		return fmt.Sprintf("rate(%d %s)", minutes, units)
	default:
		panic("unknown scheduler expression")
	}
}

// runTaskResource returns a troposphere resource that will create a lambda
// function that can be used to run an ECS task.
func runTaskResource(role interface{}) troposphere.Resource {
	return troposphere.Resource{
		Type: "AWS::Lambda::Function",
		Properties: map[string]interface{}{
			"Description": fmt.Sprintf("Lambda function to run an ECS task"),
			"Handler":     "index.handler",
			"Role":        role,
			"Runtime":     "python2.7",
			"Code": map[string]interface{}{
				"ZipFile": runTaskCode,
			},
		},
	}
}

// fmtPorts implements the fmt.Stringer interface to show a map of container
// port to host port.
type fmtPorts []twelvefactor.Port

func (p fmtPorts) String() string {
	var mappings []string
	for _, port := range p {
		mappings = append(mappings, fmt.Sprintf("%d => %d", port.Host, port.Container))
	}
	return strings.Join(mappings, ", ")
}

// cloudformationTags implements the sort.Interface interface to sort the labels
// variables by key in alphabetical order.
type cloudformationTags []*cloudformation.Tag

func (e cloudformationTags) Len() int           { return len(e) }
func (e cloudformationTags) Less(i, j int) bool { return *e[i].Key < *e[j].Key }
func (e cloudformationTags) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

// tagsFromLabels generates a list of CloudFormation tags from the labels, it
// also sorts the tags by key.
func tagsFromLabels(labels map[string]string) []*cloudformation.Tag {
	tags := cloudformationTags{}
	for k, v := range labels {
		tags = append(tags, &cloudformation.Tag{Key: aws.String(k), Value: aws.String(v)})
	}
	sort.Sort(tags)
	return tags
}

// This is a helpful function to check if any type is nil. We cannot simply
// check `v == nil` because it will return true, even if the underlying type is
// nil. Instead, we have to use reflection to check if the underlying value is
// nil.
//
// See https://play.golang.org/p/aq3DmMZ_P8
func toInterface(v interface{}) interface{} {
	if reflect.ValueOf(v).IsNil() {
		return nil
	}

	return v
}

// A simple lambda function that can be used to trigger an ecs.RunTask.
const runTaskCode = `
import boto3
import logging

logger = logging.getLogger()
logger.setLevel(logging.INFO)

ecs = boto3.client('ecs')

def handler(event, context):
  logger.info('Request Received')
  logger.info(event)

  resp = ecs.run_task(
    cluster=event['cluster'],
    taskDefinition=event['taskDefinition'],
    count=event['count'],
    startedBy=event['startedBy'])

  return map(lambda x: x['taskArn'], resp['tasks'])`
