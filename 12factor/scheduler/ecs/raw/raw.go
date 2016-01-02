// Package raw implements a StackBuilder using direct AWS API calls.
package raw

import (
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/pkg/aws/arn"
	"github.com/remind101/empire/pkg/bytesize"
)

// DefaultDelimiter is the default delimiter used to delineate between app and
// process in service names.
const DefaultDelimiter = "--"

type ecsClient interface {
	ListServicesPages(*ecs.ListServicesInput, func(*ecs.ListServicesOutput, bool) bool) error
	DeleteService(*ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error)
	RegisterTaskDefinition(*ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	CreateService(*ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error)
	UpdateService(*ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
}

// StackBuilder implements the StackBuilder interface for the ECS scheduler.
type StackBuilder struct {
	// ECS Cluster to operate within.
	Cluster string

	// Delimiter to use in the service name to delineate between app and
	// process. The zero value is DefaultDelimiter.
	Delimiter string

	// ServiceRole is the name of an IAM role to attach to ECS services that
	// have ELB's attached.
	ServiceRole string

	ecs ecsClient
}

// NewStackBuilder returns a new StackBuilder instance with an ecs client
// configured from config.
func NewStackBuilder(config client.ConfigProvider) *StackBuilder {
	return &StackBuilder{
		ecs: ecs.New(config),
	}
}

// Build creates or updates ECS services for the app.
func (b *StackBuilder) Build(manifest twelvefactor.Manifest) error {
	// TODO: Remove old services not in the manifest
	for _, process := range manifest.Processes {
		if err := b.CreateService(manifest.App, process); err != nil {
			return err
		}
	}

	return nil
}

// CreateService creates an ECS service for the Process.
func (b *StackBuilder) CreateService(app twelvefactor.App, process twelvefactor.Process) error {
	name := b.ServiceName(app, process)

	taskDefinition, err := b.RegisterTaskDefinition(app, process)
	if err != nil {
		return err
	}

	_, err = b.ecs.CreateService(&ecs.CreateServiceInput{
		Cluster:        aws.String(b.Cluster),
		DesiredCount:   aws.Int64(int64(process.DesiredCount)),
		Role:           aws.String(b.ServiceRole),
		ServiceName:    aws.String(name),
		TaskDefinition: aws.String(taskDefinition),
	})
	return err
}

func (b *StackBuilder) RegisterTaskDefinition(app twelvefactor.App, process twelvefactor.Process) (string, error) {
	family := b.TaskDefinitionName(app, process)

	var command []*string
	for _, s := range process.Command {
		ss := s
		command = append(command, &ss)
	}

	var environment []*ecs.KeyValuePair
	for k, v := range twelvefactor.ProcessEnv(app, process) {
		environment = append(environment, &ecs.KeyValuePair{
			Name:  aws.String(k),
			Value: aws.String(v),
		})
	}

	resp, err := b.ecs.RegisterTaskDefinition(&ecs.RegisterTaskDefinitionInput{
		Family: aws.String(family),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Name:        aws.String(process.Name),
				Cpu:         aws.Int64(int64(process.CPUShares)),
				Command:     command,
				Image:       aws.String(app.Image),
				Essential:   aws.Bool(true),
				Memory:      aws.Int64(int64(process.Memory / int(bytesize.MB))),
				Environment: environment,
			},
		},
	})
	if err != nil {
		return "", err
	}

	return *resp.TaskDefinition.TaskDefinitionArn, nil
}

// Iterates through all of the ECS services for this app and removes them.
func (b *StackBuilder) Remove(app string) error {
	services, err := b.Services(app)
	if err != nil {
		return err
	}

	for _, service := range services {
		// TODO: Parallelize
		if err := b.RemoveService(service); err != nil {
			return err
		}
	}

	return nil
}

// RemoveService scales an ECS service to 0, waits for it to become stable, then
// removes it.
func (b *StackBuilder) RemoveService(service string) error {
	if _, err := b.ecs.UpdateService(&ecs.UpdateServiceInput{
		Cluster:      aws.String(b.Cluster),
		Service:      aws.String(service),
		DesiredCount: aws.Int64(0),
	}); err != nil {
		return err
	}

	// TODO: Wait until https://github.com/aws/aws-sdk-go/issues/457 is
	// resolved.

	if _, err := b.ecs.DeleteService(&ecs.DeleteServiceInput{
		Cluster: aws.String(b.Cluster),
		Service: aws.String(service),
	}); err != nil {
		return err
	}

	return nil
}

// Services iterates through all of the ECS services in this cluster, and
// returns the services that are members of the given app.
func (b *StackBuilder) Services(app string) (map[string]string, error) {
	services := make(map[string]string)

	if err := b.ecs.ListServicesPages(&ecs.ListServicesInput{
		Cluster: aws.String(b.Cluster),
	}, func(resp *ecs.ListServicesOutput, lastPage bool) bool {
		for _, serviceArn := range resp.ServiceArns {
			if serviceArn == nil {
				continue
			}

			id, err := arn.ResourceID(*serviceArn)
			if err != nil {
				return false
			}

			appName, process, ok := b.split(id)
			if !ok {
				continue
			}

			if appName == app {
				services[process] = id
			}
		}

		return true
	}); err != nil {
		return nil, err
	}

	return services, nil
}

func (b *StackBuilder) ServiceName(app twelvefactor.App, process twelvefactor.Process) string {
	return strings.Join([]string{app.ID, process.Name}, b.delimiter())
}

func (b *StackBuilder) TaskDefinitionName(app twelvefactor.App, process twelvefactor.Process) string {
	return strings.Join([]string{app.ID, process.Name}, b.delimiter())
}

func (b *StackBuilder) split(service string) (app, process string, ok bool) {
	parts := strings.SplitN(service, b.delimiter(), 2)
	if len(parts) != 2 {
		return
	}
	app, process, ok = parts[0], parts[1], true
	return
}

func (b *StackBuilder) delimiter() string {
	if b.Delimiter == "" {
		return DefaultDelimiter
	}

	return b.Delimiter
}
