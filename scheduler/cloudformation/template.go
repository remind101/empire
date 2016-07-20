package cloudformation

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/troposphere"
	"github.com/remind101/empire/scheduler"
)

var (
	Ref    = troposphere.Ref
	GetAtt = troposphere.GetAtt
	Equals = troposphere.Equals
	Join   = troposphere.Join
)

const (
	// For HTTP/HTTPS/TCP services, we allocate an ELB and map it's instance port to
	// the container port. This is the port that processes within the container
	// should bind to. This value is also exposed to the container through the PORT
	// environment variable.
	ContainerPort = 8080

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
	v, err := t.Build(data.(*scheduler.App))
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
func (t *EmpireTemplate) Build(app *scheduler.App) (*troposphere.Template, error) {
	tmpl := troposphere.NewTemplate()

	tmpl.Parameters["DNS"] = troposphere.Parameter{
		Type:        "String",
		Description: "When set to `true`, CNAME's will be altered",
		Default:     "true",
	}
	tmpl.Parameters[restartParameter] = troposphere.Parameter{Type: "String"}
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
			// To save space in the template, avoid adding the
			// resources if the process is scaled down.
			if p.Instances > 0 {
				taskDefinition := t.addScheduledTask(tmpl, app, p)
				scheduledProcesses[p.Type] = taskDefinition.Name
			}
		default:
			service := t.addService(tmpl, app, p)
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

func (t *EmpireTemplate) addTaskDefinition(tmpl *troposphere.Template, app *scheduler.App, p *scheduler.Process) (troposphere.NamedResource, *ContainerDefinitionProperties) {
	key := processResourceName(p.Type)
	// The task definition that will be used to run the ECS task.
	taskDefinition := troposphere.NamedResource{
		Name: fmt.Sprintf("%sTaskDefinition", key),
	}

	cd := t.ContainerDefinition(app, p)
	containerDefinition := cloudformationContainerDefinition(cd)

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
		}
	} else {
		containerDefinition.Environment = cd.Environment
		taskDefinitionProperties = &TaskDefinitionProperties{
			Volumes: []interface{}{},
			ContainerDefinitions: []*ContainerDefinitionProperties{
				containerDefinition,
			},
		}
	}

	taskDefinition.Resource = troposphere.Resource{
		Type:       taskDefinitionType,
		Properties: taskDefinitionProperties,
	}
	tmpl.AddResource(taskDefinition)

	return taskDefinition, containerDefinition
}

func (t *EmpireTemplate) addScheduledTask(tmpl *troposphere.Template, app *scheduler.App, p *scheduler.Process) troposphere.NamedResource {
	key := processResourceName(p.Type)

	taskDefinition, _ := t.addTaskDefinition(tmpl, app, p)

	schedule := fmt.Sprintf("%sTrigger", key)
	tmpl.Resources[schedule] = troposphere.Resource{
		Type: "AWS::Events::Rule",
		Properties: map[string]interface{}{
			"Description":        fmt.Sprintf("Rule to periodically trigger the `%s` scheduled task", p.Type),
			"ScheduleExpression": scheduleExpression(p.Schedule),
			"RoleArn":            t.serviceRoleArn(),
			"State":              "ENABLED",
			"Targets": []interface{}{
				map[string]interface{}{
					"Arn":   GetAtt(runTaskFunction, "Arn"),
					"Id":    "f",
					"Input": Join("", `{"taskDefinition":"`, Ref(taskDefinition), `","count":`, Ref(scaleParameter(p.Type)), `,"cluster":"`, t.Cluster, `","startedBy": "`, app.ID, `"}`),
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

func (t *EmpireTemplate) addService(tmpl *troposphere.Template, app *scheduler.App, p *scheduler.Process) (serviceName string) {
	key := processResourceName(p.Type)

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

	loadBalancers := []map[string]interface{}{}
	if p.Exposure != nil {
		scheme := schemeInternal
		sg := t.InternalSecurityGroupID
		subnets := t.InternalSubnetIDs

		if p.Exposure.External {
			scheme = schemeExternal
			sg = t.ExternalSecurityGroupID
			subnets = t.ExternalSubnetIDs
		}

		instancePort := fmt.Sprintf("%s%dInstancePort", key, ContainerPort)
		tmpl.Resources[instancePort] = troposphere.Resource{
			Type:    "Custom::InstancePort",
			Version: "1.0",
			Properties: map[string]interface{}{
				"ServiceToken": t.CustomResourcesTopic,
			},
		}

		listeners := []map[string]interface{}{
			map[string]interface{}{
				"LoadBalancerPort": 80,
				"Protocol":         "http",
				"InstancePort":     GetAtt(instancePort, "InstancePort"),
				"InstanceProtocol": "http",
			},
		}

		if e, ok := p.Exposure.Type.(*scheduler.HTTPSExposure); ok {
			var cert interface{}
			if _, err := arn.Parse(e.Cert); err == nil {
				cert = e.Cert
			} else {
				cert = Join("", "arn:aws:iam::", Ref("AWS::AccountId"), ":server-certificate/", e.Cert)
			}

			listeners = append(listeners, map[string]interface{}{
				"LoadBalancerPort": 443,
				"Protocol":         "https",
				"InstancePort":     GetAtt(instancePort, "InstancePort"),
				"SSLCertificateId": cert,
				"InstanceProtocol": "http",
			})
		}

		portMappings = append(portMappings, &PortMappingProperties{
			ContainerPort: ContainerPort,
			HostPort:      GetAtt(instancePort, "InstancePort"),
		})
		p.Env["PORT"] = fmt.Sprintf("%d", ContainerPort)

		loadBalancer := fmt.Sprintf("%sLoadBalancer", key)
		loadBalancers = append(loadBalancers, map[string]interface{}{
			"ContainerName":    p.Type,
			"ContainerPort":    ContainerPort,
			"LoadBalancerName": Ref(loadBalancer),
		})
		tmpl.Resources[loadBalancer] = troposphere.Resource{
			Type: "AWS::ElasticLoadBalancing::LoadBalancer",
			Properties: map[string]interface{}{
				"Scheme":         scheme,
				"SecurityGroups": []string{sg},
				"Subnets":        subnets,
				"Listeners":      listeners,
				"CrossZone":      true,
				"Tags": []map[string]string{
					map[string]string{
						"Key":   "empire.app.process",
						"Value": p.Type,
					},
				},
				"ConnectionDrainingPolicy": map[string]interface{}{
					"Enabled": true,
					"Timeout": defaultConnectionDrainingTimeout,
				},
			},
		}

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

	service := fmt.Sprintf("%sService", key)
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
	tmpl.Resources[service] = troposphere.Resource{
		Type:       ecsServiceType,
		Properties: serviceProperties,
	}
	return service
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
func (t *EmpireTemplate) ContainerDefinition(app *scheduler.App, p *scheduler.Process) *ecs.ContainerDefinition {
	command := []*string{}
	for _, s := range p.Command {
		ss := s
		command = append(command, &ss)
	}

	labels := make(map[string]*string)
	for k, v := range scheduler.Labels(app, p) {
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
		Memory:           aws.Int64(int64(p.MemoryLimit / bytesize.MB)),
		Environment:      sortedEnvironment(scheduler.Env(app, p)),
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

func scheduleExpression(s scheduler.Schedule) string {
	switch v := s.(type) {
	case scheduler.CRONSchedule:
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

// Returns the name of the CloudFormation resource that should be used to create
// custom task definitions.
func taskDefinitionResourceType(app *scheduler.App) string {
	if app.Env["ECS_TASK_DEFINITION"] == "custom" {
		return "Custom::ECSTaskDefinition"
	}
	return "AWS::ECS::TaskDefinition"
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
