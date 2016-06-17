package cloudformation

import (
	"bytes"
	"crypto/sha1"
	"encoding/base64"
	"fmt"
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

// Replacement signature returns a deterministic string representing the fields
// that, when changed, require a replacement of the ECS service. This can be
// used to compare ECSServiceProperties to see if a replacement is required.
func (p *ECSServiceProperties) ReplacementSignature() string {
	buf := new(bytes.Buffer)

	s := func(v *string) string {
		if v != nil {
			return *v
		}
		return ""
	}

	d := func(v *IntValue) string {
		if v != nil {
			return fmt.Sprintf("%d", *v)
		}
		return ""
	}

	// These properties cannot be updated.
	fmt.Fprintf(buf, "ServiceName: %s\n", s(p.ServiceName))
	fmt.Fprintf(buf, "Cluster: %s\n", s(p.Cluster))
	fmt.Fprintf(buf, "Role: %s\n", s(p.Role))
	fmt.Fprintf(buf, "LoadBalancers:\n")
	for _, l := range p.LoadBalancers {
		fmt.Fprintf(buf, "  LoadBalancerName: %s\n", s(l.LoadBalancerName))
		fmt.Fprintf(buf, "  ContainerPort: %s\n", d(l.ContainerPort))
		fmt.Fprintf(buf, "  ContainerName: %s\n", s(l.ContainerName))
	}

	return buf.String()
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

		if requiresReplacement(properties, oldProperties) {
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
		serviceName = aws.String(fmt.Sprintf("%s-%s", *properties.ServiceName, postfix(properties)))
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

// Certain parameters cannot be updated on existing services, so we need to
// create a new physical resource.
func requiresReplacement(new, old *ECSServiceProperties) bool {
	return new.ReplacementSignature() != old.ReplacementSignature()
}

// It's important that the postfix we append to the service name
// is deterministic based on the non replaceable fields of the
// service, otherwise ClientToken has no effect on idempotency.
func postfix(p *ECSServiceProperties) string {
	h := sha1.New()
	h.Write([]byte(p.ReplacementSignature()))
	b := h.Sum(nil)
	return strings.Replace(base64.URLEncoding.EncodeToString(b), "=", "", -1)
}
