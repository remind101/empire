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

		if canUpdateService(properties, oldProperties) {
			_, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
				Service:        aws.String(id),
				Cluster:        properties.Cluster,
				DesiredCount:   properties.DesiredCount.Value(),
				TaskDefinition: properties.TaskDefinition,
			})
			return id, nil, err
		}

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

// We currently only support updating the task definition and desired count.
func canUpdateService(new, old *ECSServiceProperties) bool {
	eq := reflect.DeepEqual

	if !eq(new.Cluster, old.Cluster) {
		return false
	}

	if !eq(new.Role, old.Role) {
		return false
	}

	if !eq(new.ServiceName, old.ServiceName) {
		return false
	}

	if !eq(new.LoadBalancers, old.LoadBalancers) {
		return false
	}

	return true
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
