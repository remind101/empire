package cloudformation

import (
	"fmt"
	"strings"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/pkg/base62"
	"github.com/remind101/empire/pkg/hashstructure"
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
	DesiredCount   *IntValue      `hash:"ignore"`
	LoadBalancers  []LoadBalancer `hash:"set"`
	Role           *string
	TaskDefinition *string `hash:"ignore"`
}

// Hash returns the Sum64 hash of the properties that, when updated, would
// require a replacement.
func (p *ECSServiceProperties) Hash() (uint64, error) {
	h, err := hashstructure.Hash(p, nil)
	if err != nil {
		err = fmt.Errorf("error hashing properties: %v", err)
	}
	return h, err
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

	switch req.RequestType {
	case Create:
		id, err := p.create(ctx, hashRequest(req), properties)
		return id, nil, err
	case Delete:
		id := req.PhysicalResourceId
		err := p.delete(ctx, aws.String(id), properties.Cluster)
		return id, nil, err
	case Update:
		id := req.PhysicalResourceId

		// Compare the hash of the properties to determine if a
		// replacement is required.
		replace, err := requiresReplacement(properties, oldProperties)
		if err != nil {
			return id, nil, fmt.Errorf("error hashing properties: %v", err)
		}

		if replace {
			// If we can't update the service, we'll need to create a new
			// one, and destroy the old one.
			oldId := id
			id, err := p.create(ctx, hashRequest(req), properties)
			if err != nil {
				return oldId, nil, err
			}

			// There's no need to delete the old service here, since
			// CloudFormation will send us a DELETE request for the old
			// service.

			return id, nil, err
		}

		_, err = p.ecs.UpdateService(&ecs.UpdateServiceInput{
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

func (p *ECSServiceResource) create(ctx context.Context, clientToken string, properties *ECSServiceProperties) (string, error) {
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
		s, err := postfix(properties)
		if err != nil {
			return "", fmt.Errorf("error hashing properties: %v", err)
		}
		serviceName = aws.String(fmt.Sprintf("%s-%s", *properties.ServiceName, s))
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
		return "", fmt.Errorf("error creating service: %v", err)
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
		return *arn, ctx.Err()
	}

	return *arn, nil
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

// It's important that the postfix we append to the service name
// is deterministic based on the non replaceable fields of the
// service, otherwise ClientToken has no effect on idempotency.
func postfix(p *ECSServiceProperties) (string, error) {
	h, err := p.Hash()
	if err != nil {
		return "", err
	}
	return base62.Encode(h), nil
}
