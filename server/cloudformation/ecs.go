package cloudformation

import (
	"fmt"
	"reflect"
	"strings"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/pkg/reporter"
)

type ecsClient interface {
	CreateService(*ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error)
	DeleteService(*ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	WaitUntilServicesStable(*ecs.DescribeServicesInput) error
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

func (p *ECSServiceResource) Provision(ctx context.Context, req Request) (string, interface{}, error) {
	properties := req.ResourceProperties.(*ECSServiceProperties)
	oldProperties := req.OldResourceProperties.(*ECSServiceProperties)
	data := make(map[string]string)

	switch req.RequestType {
	case Create:
		id, deploymentId, err := p.create(ctx, hashRequest(req), properties)
		if err != nil {
			data["DeploymentId"] = deploymentId
		}
		return id, data, err
	case Delete:
		id := req.PhysicalResourceId
		err := p.delete(ctx, aws.String(id), properties.Cluster)
		return id, nil, err
	case Update:
		id := req.PhysicalResourceId

		if requiresReplacement(properties, oldProperties) {
			// If we can't update the service, we'll need to create a new
			// one, and destroy the old one.
			oldId := id
			id, deploymentId, err := p.create(ctx, hashRequest(req), properties)
			if err != nil {
				return oldId, nil, err
			}
			data["DeploymentId"] = deploymentId

			// There's no need to delete the old service here, since
			// CloudFormation will send us a DELETE request for the old
			// service.

			return id, data, err
		}

		resp, err := p.ecs.UpdateService(&ecs.UpdateServiceInput{
			Service:        aws.String(id),
			Cluster:        properties.Cluster,
			DesiredCount:   properties.DesiredCount.Value(),
			TaskDefinition: properties.TaskDefinition,
		})
		if err == nil {
			primary, err := getPrimaryDeployment(resp.Service)
			if err == nil {
				data["DeploymentId"] = *primary.Id
			}
		}
		return id, data, err
	default:
		return "", nil, fmt.Errorf("%s is not supported", req.RequestType)
	}
}

func (p *ECSServiceResource) create(ctx context.Context, clientToken string, properties *ECSServiceProperties) (string, string, error) {
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
		serviceName = aws.String(fmt.Sprintf("%s-%s", *properties.ServiceName, clientToken))
	}

	resp, err := p.ecs.CreateService(&ecs.CreateServiceInput{
		ClientToken:    aws.String(clientToken),
		ServiceName:    serviceName,
		Cluster:        properties.Cluster,
		DesiredCount:   properties.DesiredCount.Value(),
		Role:           properties.Role,
		TaskDefinition: properties.TaskDefinition,
		LoadBalancers:  loadBalancers,
	})
	if err != nil {
		return "", "", fmt.Errorf("error creating service: %v", err)
	}

	primaryDeployment, err := getPrimaryDeployment(resp.Service)
	if err != nil {
		return "", "", fmt.Errorf("error retrieving primary deployment: %v", err)
	}

	arn := resp.Service.ServiceArn

	stabilized := make(chan struct{})
	go func() {
		if err := p.ecs.WaitUntilServicesStable(&ecs.DescribeServicesInput{
			Cluster:  properties.Cluster,
			Services: []*string{arn},
		}); err != nil {
			// We're ignoring this error, because the service was created,
			// and if the service doesn't stabilize, it's better to just let
			// the stack finish creating than rolling back.
			reporter.Report(ctx, err)
		}
		close(stabilized)
	}()

	select {
	case <-stabilized:
	case <-ctx.Done():
		return *arn, *primaryDeployment.Id, ctx.Err()
	}

	return *arn, *primaryDeployment.Id, nil
}

func (p *ECSServiceResource) delete(ctx context.Context, service, cluster *string) error {
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

func getPrimaryDeployment(service *ecs.Service) (deployment *ecs.Deployment, err error) {
	for _, d := range service.Deployments {
		if d.Status != nil && *d.Status == "PRIMARY" {
			deployment = d
			break
		}
	}
	if deployment == nil {
		err = fmt.Errorf("no primary deployment available")
	}
	return
}
