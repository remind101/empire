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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/arn"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/scheduler"
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

	// The name of the ECS Service IAM role.
	ServiceRole string

	// The ARN of the SNS topic to provision instance ports.
	CustomResourcesTopic string

	LogConfiguration *ecs.LogConfiguration
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
func (t *EmpireTemplate) Build(app *scheduler.App) (interface{}, error) {
	parameters := map[string]interface{}{
		"DNS": map[string]string{
			"Type":        "String",
			"Description": "When set to `true`, CNAME's will be altered",
			"Default":     "true",
		},
		restartParameter: map[string]string{
			"Type": "String",
		},
	}
	conditions := map[string]interface{}{
		"DNSCondition": map[string]interface{}{
			"Fn::Equals": []interface{}{
				map[string]string{
					"Ref": "DNS",
				},
				"true",
			},
		},
	}
	resources := map[string]interface{}{}
	outputs := map[string]interface{}{
		"Release": map[string]interface{}{
			"Value": app.Release,
		},
		"EmpireVersion": map[string]interface{}{
			"Value": empire.Version,
		},
	}

	serviceMappings := []map[string]interface{}{}
	deploymentMappings := []map[string]interface{}{}

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
	if app.Env["ECS_SERVICE"] == "standard" {
		ecsServiceType = "AWS::ECS::Service"
	}

	for _, p := range app.Processes {
		cd := t.ContainerDefinition(app, p)

		key := processResourceName(p.Type)

		parameters[scaleParameter(p.Type)] = map[string]string{
			"Type": "String",
		}

		portMappings := []map[string]interface{}{}

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
			resources[instancePort] = map[string]interface{}{
				"Type":    "Custom::InstancePort",
				"Version": "1.0",
				"Properties": map[string]interface{}{
					"ServiceToken": t.CustomResourcesTopic,
				},
			}

			listeners := []map[string]interface{}{
				map[string]interface{}{
					"LoadBalancerPort": 80,
					"Protocol":         "http",
					"InstancePort": map[string][]string{
						"Fn::GetAtt": []string{
							instancePort,
							"InstancePort",
						},
					},
					"InstanceProtocol": "http",
				},
			}

			if e, ok := p.Exposure.Type.(*scheduler.HTTPSExposure); ok {
				var cert interface{}
				if _, err := arn.Parse(e.Cert); err == nil {
					cert = e.Cert
				} else {
					cert = map[string]interface{}{
						"Fn::Join": []interface{}{
							"",
							[]interface{}{"arn:aws:iam::", map[string]string{"Ref": "AWS::AccountId"}, ":server-certificate/", e.Cert},
						},
					}
				}

				listeners = append(listeners, map[string]interface{}{
					"LoadBalancerPort": 443,
					"Protocol":         "https",
					"InstancePort": map[string][]string{
						"Fn::GetAtt": []string{
							instancePort,
							"InstancePort",
						},
					},
					"SSLCertificateId": cert,
					"InstanceProtocol": "http",
				})
			}

			portMappings = append(portMappings, map[string]interface{}{
				"ContainerPort": ContainerPort,
				"HostPort": map[string][]string{
					"Fn::GetAtt": []string{
						instancePort,
						"InstancePort",
					},
				},
			})
			cd.Environment = append(cd.Environment, &ecs.KeyValuePair{
				Name:  aws.String("PORT"),
				Value: aws.String(fmt.Sprintf("%d", ContainerPort)),
			})

			loadBalancer := fmt.Sprintf("%sLoadBalancer", key)
			loadBalancers = append(loadBalancers, map[string]interface{}{
				"ContainerName": p.Type,
				"ContainerPort": ContainerPort,
				"LoadBalancerName": map[string]string{
					"Ref": loadBalancer,
				},
			})
			resources[loadBalancer] = map[string]interface{}{
				"Type": "AWS::ElasticLoadBalancing::LoadBalancer",
				"Properties": map[string]interface{}{
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
				resources["CNAME"] = map[string]interface{}{
					"Type":      "AWS::Route53::RecordSet",
					"Condition": "DNSCondition",
					"Properties": map[string]interface{}{
						"HostedZoneId": *t.HostedZone.Id,
						"Name":         fmt.Sprintf("%s.%s", app.Name, *t.HostedZone.Name),
						"Type":         "CNAME",
						"TTL":          defaultCNAMETTL,
						"ResourceRecords": []map[string][]string{
							map[string][]string{
								"Fn::GetAtt": []string{loadBalancer, "DNSName"},
							},
						},
					},
				}
			}
		}

		labels := map[string]interface{}{}
		for k, v := range cd.DockerLabels {
			labels[k] = v
		}
		labels["cloudformation.restart-key"] = map[string]string{"Ref": restartParameter}

		taskDefinition := fmt.Sprintf("%sTaskDefinition", key)
		containerDefinition := map[string]interface{}{
			"Name":         *cd.Name,
			"Command":      cd.Command,
			"Cpu":          *cd.Cpu,
			"Image":        *cd.Image,
			"Essential":    *cd.Essential,
			"Memory":       *cd.Memory,
			"Environment":  cd.Environment,
			"PortMappings": portMappings,
			"DockerLabels": labels,
			"Ulimits":      cd.Ulimits,
		}
		if cd.LogConfiguration != nil {
			containerDefinition["LogConfiguration"] = cd.LogConfiguration
		}
		resources[taskDefinition] = map[string]interface{}{
			"Type": "AWS::ECS::TaskDefinition",
			"Properties": map[string]interface{}{
				"ContainerDefinitions": []interface{}{
					containerDefinition,
				},
				"Volumes": []interface{}{},
			},
		}

		service := fmt.Sprintf("%s", key)
		serviceProperties := map[string]interface{}{
			"Cluster": t.Cluster,
			"DesiredCount": map[string]string{
				"Ref": scaleParameter(p.Type),
			},
			"LoadBalancers": loadBalancers,
			"TaskDefinition": map[string]string{
				"Ref": taskDefinition,
			},
		}
		if ecsServiceType == "Custom::ECSService" {
			// It's not possible to change the type of a resource,
			// so we have to change the name of the service resource
			// to something different (just append "Service").
			service = fmt.Sprintf("%sService", service)
			serviceProperties["ServiceName"] = fmt.Sprintf("%s-%s", app.Name, p.Type)
			serviceProperties["ServiceToken"] = t.CustomResourcesTopic
		}
		if len(loadBalancers) > 0 {
			serviceProperties["Role"] = t.ServiceRole
		}
		serviceMappings = append(serviceMappings, map[string]interface{}{
			"Fn::Join": []interface{}{
				"=",
				[]interface{}{p.Type, map[string]string{"Ref": service}},
			},
		})
		deploymentMappings = append(deploymentMappings, map[string]interface{}{
			"Fn::Join": []interface{}{
				"=",
				[]interface{}{p.Type, map[string][]string{
					"Fn::GetAtt": []string{
						service,
						"DeploymentId",
					},
				}},
			},
		})
		resources[service] = map[string]interface{}{
			"Type":       ecsServiceType,
			"Properties": serviceProperties,
		}

	}

	outputs[servicesOutput] = map[string]interface{}{
		"Value": map[string]interface{}{
			"Fn::Join": []interface{}{
				",",
				serviceMappings,
			},
		},
	}
	outputs[deploymentsOutput] = map[string]interface{}{
		"Value": map[string]interface{}{
			"Fn::Join": []interface{}{
				",",
				deploymentMappings,
			},
		},
	}

	return map[string]interface{}{
		"Parameters": parameters,
		"Conditions": conditions,
		"Resources":  resources,
		"Outputs":    outputs,
	}, nil
}

// envByKey implements the sort.Interface interface to sort the environment
// variables by key in alphabetical order.
type envByKey []*ecs.KeyValuePair

func (e envByKey) Len() int           { return len(e) }
func (e envByKey) Less(i, j int) bool { return *e[i].Name < *e[j].Name }
func (e envByKey) Swap(i, j int)      { e[i], e[j] = e[j], e[i] }

// ContainerDefinition generates an ECS ContainerDefinition for a process.
func (t *EmpireTemplate) ContainerDefinition(app *scheduler.App, p *scheduler.Process) *ecs.ContainerDefinition {
	command := []*string{}
	for _, s := range p.Command {
		ss := s
		command = append(command, &ss)
	}

	environment := envByKey{}
	for k, v := range scheduler.Env(app, p) {
		environment = append(environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	sort.Sort(environment)

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
		Environment:      environment,
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
