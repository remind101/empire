package cloudformation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/scheduler"
)

const (

	// For HTTP/HTTPS/TCP services, we allocate an ELB and map it's instance port to
	// the container port. This is the port that processes within the container
	// should bind to. Tihs value is also exposed to the container through the PORT
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

	LogConfiguration *ecs.LogConfiguration
}

// Execute builds the template, and writes it to w.
func (t *EmpireTemplate) Execute(w io.Writer, data interface{}) error {
	v, err := t.Build(data.(*scheduler.App))
	if err != nil {
		return err
	}

	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}

	_, err = io.Copy(w, bytes.NewReader(raw))
	return err
}

// Build builds a Go representation of a CloudFormation template for the app.
func (t *EmpireTemplate) Build(app *scheduler.App) (interface{}, error) {
	parameters := map[string]interface{}{}
	resources := map[string]interface{}{}
	outputs := map[string]interface{}{}

	serviceRole := "ServiceRole" // TODO: Build a service role.
	parameters["ServiceRole"] = map[string]string{
		"Type":    "String",
		"Default": "ecsServiceRole",
	}

	for _, p := range app.Processes {
		ulimits := []map[string]interface{}{}
		if p.Nproc != 0 {
			ulimits = []map[string]interface{}{
				map[string]interface{}{
					"Name":      "nproc",
					"SoftLimit": p.Nproc,
					"HardLimit": p.Nproc,
				},
			}
		}

		portMappings := []map[string]int64{}

		labels := p.Labels
		if labels == nil {
			labels = make(map[string]string)
		}

		environment := []map[string]string{}
		for k, v := range p.Env {
			environment = append(environment, map[string]string{
				"Name":  k,
				"Value": v,
			})
		}

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

			instancePort := int64(9000) // TODO: Allocate a port
			listeners := []map[string]interface{}{
				map[string]interface{}{
					"LoadBalancerPort": 80,
					"Protocol":         "http",
					"InstancePort":     instancePort,
					"InstanceProtocol": "http",
				},
			}

			if e, ok := p.Exposure.Type.(*scheduler.HTTPSExposure); ok {
				listeners = append(listeners, map[string]interface{}{
					"LoadBalancerPort": 80,
					"Protocol":         "http",
					"InstancePort":     instancePort,
					"SSLCertificateId": e.Cert,
					"InstanceProtocol": "http",
				})
			}

			portMappings = append(portMappings, map[string]int64{
				"ContainerPort": ContainerPort,
				"HostPort":      instancePort,
			})
			environment = append(environment, map[string]string{
				"Name":  "PORT",
				"Value": fmt.Sprintf("%d", ContainerPort),
			})

			loadBalancer := fmt.Sprintf("%sLoadBalancer", p.Type)
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
					"Type": "AWS::Route53::RecordSet",
					"Properties": map[string]interface{}{
						"HostedZoneId": *t.HostedZone.Id,
						"Name":         fmt.Sprintf("%s.%s", app.Name, *t.HostedZone.Name),
						"Type":         "CNAME",
						"TTL":          defaultCNAMETTL,
						"ResourceRecords": []map[string]string{
							map[string]string{
								"Ref": loadBalancer,
							},
						},
					},
				}
			}
		}

		taskDefinition := fmt.Sprintf("%sTaskDefinition", p.Type)
		containerDefinition := map[string]interface{}{
			"Name":         p.Type,
			"Command":      p.Command,
			"Cpu":          p.CPUShares,
			"Image":        p.Image.String(),
			"Essential":    true,
			"Memory":       p.MemoryLimit / bytesize.MB,
			"Environment":  environment,
			"PortMappings": portMappings,
			"DockerLabels": labels,
			"Ulimits":      ulimits,
		}
		if t.LogConfiguration != nil {
			containerDefinition["LogConfiguration"] = t.LogConfiguration
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

		service := fmt.Sprintf("%s", p.Type)
		serviceProperties := map[string]interface{}{
			"Cluster":       t.Cluster,
			"DesiredCount":  p.Instances,
			"LoadBalancers": loadBalancers,
			"TaskDefinition": map[string]string{
				"Ref": taskDefinition,
			},
		}
		if len(loadBalancers) > 0 {
			serviceProperties["Role"] = map[string]string{
				"Ref": serviceRole,
			}
		}
		resources[service] = map[string]interface{}{
			"Type": "AWS::ECS::Service",
			"Metadata": &serviceMetadata{
				Name: p.Type,
			},
			"Properties": serviceProperties,
		}

	}

	return map[string]interface{}{
		"Parameters": parameters,
		"Resources":  resources,
		"Outputs":    outputs,
	}, nil
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
