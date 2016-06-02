package cloudformation

import (
	"fmt"
	"math/rand"
	"reflect"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
)

type ecsClient interface {
	CreateService(*ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error)
	DeleteService(*ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	RegisterTaskDefinition(*ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	DeregisterTaskDefinition(*ecs.DeregisterTaskDefinitionInput) (*ecs.DeregisterTaskDefinitionOutput, error)
}

type LoadBalancer struct {
	ContainerName    *string
	ContainerPort    *IntValue
	LoadBalancerName *string
}

// ECSServiceProperties represents the properties for the Custom::ECSService
// resource.
type ECSServiceProperties struct {
	ServiceName    *string
	Cluster        *string
	DesiredCount   *IntValue
	LoadBalancers  []LoadBalancer
	Role           *string
	TaskDefinition *string
}

// ECSServiceResource is a Provisioner that creates and updates ECS services.
type ECSServiceResource struct {
	ecs ecsClient

	// postfix returns a string that should be appended when creating new
	// ecs services.
	postfix func() string
}

func (p *ECSServiceResource) Properties() interface{} {
	return &ECSServiceProperties{}
}

func (p *ECSServiceResource) Provision(req Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSServiceProperties)
	oldProperties := req.OldResourceProperties.(*ECSServiceProperties)

	switch req.RequestType {
	case Create:
		id, err := p.create(properties)
		return id, nil, err
	case Delete:
		id := req.PhysicalResourceId
		err := p.delete(aws.String(id), properties.Cluster)
		return id, nil, err
	case Update:
		id := req.PhysicalResourceId

		if requiresReplacement(properties, oldProperties) {
			// If we can't update the service, we'll need to create a new
			// one, and destroy the old one.
			oldId := id
			id, err := p.create(properties)
			if err != nil {
				return oldId, nil, err
			}

			// There's no need to delete the old service here, since
			// CloudFormation will send us a DELETE request for the old
			// service.

			return id, nil, err
		}

		_, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
			Service:        aws.String(id),
			Cluster:        properties.Cluster,
			DesiredCount:   properties.DesiredCount.Value(),
			TaskDefinition: properties.TaskDefinition,
		})
		return id, nil, err
	default:
		return "", nil, fmt.Errorf("%s is not supported", req.RequestType)
	}
}

func (p *ECSServiceResource) create(properties *ECSServiceProperties) (string, error) {
	var loadBalancers []*ecs.LoadBalancer
	for _, v := range properties.LoadBalancers {
		loadBalancers = append(loadBalancers, &ecs.LoadBalancer{
			ContainerName:    v.ContainerName,
			ContainerPort:    v.ContainerPort.Value(),
			LoadBalancerName: v.LoadBalancerName,
		})
	}

	var serviceName *string
	if properties.ServiceName != nil {
		serviceName = aws.String(*properties.ServiceName + p.postfix())
	}

	resp, err := p.ecs.CreateService(&ecs.CreateServiceInput{
		ServiceName:    serviceName,
		Cluster:        properties.Cluster,
		DesiredCount:   properties.DesiredCount.Value(),
		Role:           properties.Role,
		TaskDefinition: properties.TaskDefinition,
		LoadBalancers:  loadBalancers,
	})
	if err != nil {
		return "", fmt.Errorf("error creating service: %v", err)
	}

	return *resp.Service.ServiceArn, nil
}

func (p *ECSServiceResource) delete(service, cluster *string) error {
	// We have to scale the service down to 0, before we're able to
	// destroy it.
	if _, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
		Service:      service,
		Cluster:      cluster,
		DesiredCount: aws.Int64(0),
	}); err != nil {
		if err, ok := err.(awserr.Error); ok && strings.Contains(err.Message(), "Service was not ACTIVE") {
			// If the service is not active, it was probably manually
			// removed already.
			return nil
		}
		return fmt.Errorf("error scaling service to 0: %v", err)
	}

	if _, err := p.ecs.DeleteService(&ecs.DeleteServiceInput{
		Service: service,
		Cluster: cluster,
	}); err != nil {
		return fmt.Errorf("error deleting service: %v", err)
	}

	return nil
}

type PortMapping struct {
	ContainerPort *IntValue
	HostPort      *IntValue
}

type Ulimit struct {
	Name      *string
	HardLimit *IntValue
	SoftLimit *IntValue
}

type ContainerDefinition struct {
	Name             *string
	Command          []*string
	Cpu              *IntValue
	Image            *string
	Essential        *string
	Memory           *IntValue
	PortMappings     []*PortMapping
	DockerLabels     map[string]*string
	Ulimits          []*Ulimit
	Environment      []*string
	LogConfiguration *ecs.LogConfiguration
}

// TaskDefinitionProperties are properties passed to the
// Custom::ECSTaskDefinition custom resource.
type ECSTaskDefinitionProperties struct {
	Family               *string
	ContainerDefinitions []ContainerDefinition
}

// ECSTaskDefinitionResource is a custom resource that provisions ECS task
// definitions.
type ECSTaskDefinitionResource struct {
	ecs ecsClient

	// postfix returns a string that should be appended when creating new
	// ecs services.
	postfix func() string
}

func (p *ECSTaskDefinitionResource) Properties() interface{} {
	return &ECSTaskDefinitionProperties{}
}

func (p *ECSTaskDefinitionResource) Provision(req Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSTaskDefinitionProperties)

	switch req.RequestType {
	case Create:
		id, err := p.create(properties)
		return id, nil, err
	case Delete:
		id := req.PhysicalResourceId
		err := p.delete(id)
		return id, nil, err
	case Update:
		id, err := p.create(properties)
		return id, nil, err
	default:
		return "", nil, fmt.Errorf("%s is not supported", req.RequestType)
	}
}

func (p *ECSTaskDefinitionResource) create(properties *ECSTaskDefinitionProperties) (string, error) {
	var containerDefinitions []*ecs.ContainerDefinition
	for _, c := range properties.ContainerDefinitions {
		// TODO: Fetch, and unencrypt environment.
		var (
			env          []*ecs.KeyValuePair
			ulimits      []*ecs.Ulimit
			portMappings []*ecs.PortMapping
			essential    *bool
		)

		for _, u := range c.Ulimits {
			ulimits = append(ulimits, &ecs.Ulimit{
				Name:      u.Name,
				HardLimit: u.HardLimit.Value(),
				SoftLimit: u.SoftLimit.Value(),
			})
		}

		for _, m := range c.PortMappings {
			portMappings = append(portMappings, &ecs.PortMapping{
				ContainerPort: m.ContainerPort.Value(),
				HostPort:      m.HostPort.Value(),
			})
		}

		if c.Essential != nil {
			essential = aws.Bool(*c.Essential == "true")
		}

		containerDefinitions = append(containerDefinitions, &ecs.ContainerDefinition{
			Name:             c.Name,
			Command:          c.Command,
			Cpu:              c.Cpu.Value(),
			Image:            c.Image,
			Essential:        essential,
			Memory:           c.Memory.Value(),
			PortMappings:     portMappings,
			DockerLabels:     c.DockerLabels,
			Ulimits:          ulimits,
			LogConfiguration: c.LogConfiguration,
			Environment:      env,
		})
	}

	var family *string
	if properties.Family != nil {
		family = aws.String(*properties.Family + p.postfix())
	}
	resp, err := p.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family:               family,
		ContainerDefinitions: containerDefinitions,
	})
	if err != nil {
		return "", fmt.Errorf("error creating task definition: %v", err)
	}
	return *resp.TaskDefinition.TaskDefinitionArn, nil
}

func (p *ECSTaskDefinitionResource) delete(arn string) error {
	// We're ignoring errors here because we really don't care if this
	// fails.
	p.ecs.DeregisterTaskDefinition(&ecs.DeregisterTaskDefinitionInput{
		TaskDefinition: aws.String(arn),
	})
	return nil
}

// Certain parameters cannot be updated on existing services, so we need to
// create a new physical resource.
func requiresReplacement(new, old *ECSServiceProperties) bool {
	eq := reflect.DeepEqual

	if !eq(new.Cluster, old.Cluster) {
		return true
	}

	if !eq(new.Role, old.Role) {
		return true
	}

	if !eq(new.ServiceName, old.ServiceName) {
		return true
	}

	if !eq(new.LoadBalancers, old.LoadBalancers) {
		return true
	}

	return false
}

var letters = []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ123456789")

// Generates a random 12 character string (similar to how standard
// CloudFormation works).
func postfix() string {
	n := 12
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return fmt.Sprintf("-%s", string(b))
}
