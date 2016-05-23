package cloudformation

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/aws/aws-sdk-go/aws"
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
}

func (p *ECSServiceResource) Properties() interface{} {
	return &ECSServiceProperties{}
}

func (p *ECSServiceResource) Provision(req Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSServiceProperties)
	oldProperties := req.OldResourceProperties.(*ECSServiceProperties)

	switch req.RequestType {
	case Create:
		var loadBalancers []*ecs.LoadBalancer
		for _, v := range properties.LoadBalancers {
			loadBalancers = append(loadBalancers, &ecs.LoadBalancer{
				ContainerName:    v.ContainerName,
				ContainerPort:    v.ContainerPort.Value(),
				LoadBalancerName: v.LoadBalancerName,
			})
		}

		resp, err := p.ecs.CreateService(&ecs.CreateServiceInput{
			ServiceName:    properties.ServiceName,
			Cluster:        properties.Cluster,
			DesiredCount:   properties.DesiredCount.Value(),
			Role:           properties.Role,
			TaskDefinition: properties.TaskDefinition,
			LoadBalancers:  loadBalancers,
		})
		if err != nil {
			return "", nil, err
		}

		return *resp.Service.ServiceArn, nil, nil
	case Delete:
		id := req.PhysicalResourceId

		// We have to scale the service down to 0, before we're able to
		// destroy it.
		_, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
			Service:      aws.String(id),
			Cluster:      properties.Cluster,
			DesiredCount: aws.Int64(0),
		})
		if err != nil {
			return id, nil, err
		}

		_, err = p.ecs.DeleteService(&ecs.DeleteServiceInput{
			Service: aws.String(id),
			Cluster: properties.Cluster,
		})

		return id, nil, err
	case Update:
		id := req.PhysicalResourceId

		if err := canUpdateService(properties, oldProperties); err != nil {
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

// We currently only support updating the task definition and desired count.
func canUpdateService(new, old *ECSServiceProperties) error {
	eq := reflect.DeepEqual

	if !eq(new.Cluster, old.Cluster) {
		return errors.New("cannot update cluster")
	}

	if !eq(new.Role, old.Role) {
		return errors.New("cannot update role")
	}

	if !eq(new.ServiceName, old.ServiceName) {
		return errors.New("cannot update service name")
	}

	if !eq(new.LoadBalancers, old.LoadBalancers) {
		return errors.New("cannot update load balancers")
	}

	return nil
}
