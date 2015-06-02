// Package ecsutil is a layer on top of Amazon ECS to provide an app aware ECS
// client.
package ecsutil

import (
	"strings"

	"golang.org/x/net/context"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/empire/pkg/arn"
)

var (
	// DefaultDelimiter is the default delimiter used to separate the app name from
	// the service type.
	DefaultDelimiter = "--"
)

// Client is an app aware ECS client.
type Client struct {
	// client used to interact with the ecs API.
	ECS

	// The delimiter to use to separate app name from service type. Zero
	// value is the DefaultDelimiter.
	Delimiter string
}

// NewClient returns a new Client instance using the provided aws.Config.
func NewClient(config *aws.Config) *Client {
	return &Client{
		ECS: newECSClient(config),
	}
}

// CreateAppService creates a new ecs service for the app.
func (c *Client) CreateAppService(ctx context.Context, app string, input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	input.ServiceName = c.prefix(app, input.ServiceName)
	input.TaskDefinition = c.prefix(app, input.TaskDefinition)
	return c.ECS.CreateService(ctx, input)
}

// DeleteAppService deletes the service for the app.
func (c *Client) DeleteAppService(ctx context.Context, app string, input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	input.Service = c.prefix(app, input.Service)
	return c.ECS.DeleteService(ctx, input)
}

// UpdateAppService updates the service for the app.
func (c *Client) UpdateAppService(ctx context.Context, app string, input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	input.Service = c.prefix(app, input.Service)
	input.TaskDefinition = c.prefix(app, input.TaskDefinition)
	return c.ECS.UpdateService(ctx, input)
}

// RegisterAppTaskDefinition register a task definition for the app.
func (c *Client) RegisterAppTaskDefinition(ctx context.Context, app string, input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	input.Family = c.prefix(app, input.Family)
	return c.ECS.RegisterTaskDefinition(ctx, input)
}

// ListAppTasks lists all the tasks for the app.
func (c *Client) ListAppTasks(ctx context.Context, app string, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	var arns []*string

	resp, err := c.ListAppServices(ctx, app, &ecs.ListServicesInput{
		Cluster: input.Cluster,
	})
	if err != nil {
		return nil, err
	}

	for _, s := range resp.ServiceARNs {
		id, err := arn.ResourceID(*s)
		if err != nil {
			return nil, err
		}

		t, err := c.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster:     input.Cluster,
			ServiceName: aws.String(id),
		})
		if err != nil {
			return nil, err
		}

		if len(t.TaskARNs) == 0 {
			continue
		}

		arns = append(arns, t.TaskARNs...)
	}

	return &ecs.ListTasksOutput{
		TaskARNs: arns,
	}, nil
}

// ListAppServices lists all services for the app.
func (c *Client) ListAppServices(ctx context.Context, app string, input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	resp, err := c.ListServices(ctx, input)
	if err != nil {
		return resp, err
	}

	var arns []*string
	for _, a := range resp.ServiceARNs {
		if a == nil {
			continue
		}

		id, err := arn.ResourceID(*a)
		if err != nil {
			return resp, err
		}

		appName, _ := c.split(&id)

		if appName == app {
			arns = append(arns, a)
		}
	}

	return &ecs.ListServicesOutput{
		ServiceARNs: arns,
	}, nil
}

func (c *Client) delimiter() string {
	if c.Delimiter == "" {
		return DefaultDelimiter
	}

	return c.Delimiter
}

func (c *Client) prefix(app string, original *string) *string {
	if original == nil {
		return nil
	}

	return aws.String(app + c.delimiter() + *original)
}

func (c *Client) split(original *string) (app string, other *string) {
	if original == nil {
		return "", nil
	}

	parts := strings.Split(*original, c.delimiter())
	if len(parts) < 2 {
		return parts[0], nil
	}

	return parts[0], &parts[1]
}
