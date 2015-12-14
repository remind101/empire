// Package ecsutil is a layer on top of Amazon ECS to provide an app aware ECS
// client.
package ecsutil

import (
	"strings"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/pkg/arn"
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
func NewClient(p client.ConfigProvider) *Client {
	return &Client{
		ECS: newECSClient(p),
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
func (c *Client) ListAppTasks(ctx context.Context, appID string, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	var taskArns []*string

	resp, err := c.listAppServiceTasks(ctx, appID, input)
	if err != nil {
		return nil, err
	}
	taskArns = append(taskArns, resp.TaskArns...)

	resp, err = c.listAppRunTasks(ctx, appID, input)
	if err != nil {
		return nil, err
	}
	taskArns = append(taskArns, resp.TaskArns...)

	return &ecs.ListTasksOutput{
		TaskArns: taskArns,
	}, nil
}

func (c *Client) listAppRunTasks(ctx context.Context, appID string, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	var taskArns []*string

	if err := c.ListTasksPages(ctx, &ecs.ListTasksInput{
		Cluster:   input.Cluster,
		StartedBy: aws.String(appID),
	}, func(resp *ecs.ListTasksOutput, lastPage bool) bool {
		taskArns = append(taskArns, resp.TaskArns...)
		return true
	}); err != nil {
		return nil, err
	}

	return &ecs.ListTasksOutput{
		TaskArns: taskArns,
	}, nil
}

func (c *Client) listAppServiceTasks(ctx context.Context, appID string, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	var arns []*string

	resp, err := c.ListAppServices(ctx, appID, &ecs.ListServicesInput{
		Cluster: input.Cluster,
	})
	if err != nil {
		return nil, err
	}

	// TODO(ejholmes): Parallelize the calls to list the tasks.
	for _, s := range resp.ServiceArns {
		id, err := arn.ResourceID(*s)
		if err != nil {
			return nil, err
		}

		var taskArns []*string
		if err := c.ListTasksPages(ctx, &ecs.ListTasksInput{
			Cluster:     input.Cluster,
			ServiceName: aws.String(id),
		}, func(resp *ecs.ListTasksOutput, lastPage bool) bool {
			taskArns = append(taskArns, resp.TaskArns...)
			return true
		}); err != nil {
			return nil, err
		}

		if len(taskArns) == 0 {
			continue
		}

		arns = append(arns, taskArns...)
	}

	return &ecs.ListTasksOutput{
		TaskArns: arns,
	}, nil
}

// ListAppServices lists all services for the app.
func (c *Client) ListAppServices(ctx context.Context, appID string, input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	var serviceArns []*string
	if err := c.ListServicesPages(ctx, input, func(resp *ecs.ListServicesOutput, lastPage bool) bool {
		serviceArns = append(serviceArns, resp.ServiceArns...)
		return true
	}); err != nil {
		return nil, err
	}

	var arns []*string
	for _, a := range serviceArns {
		if a == nil {
			continue
		}

		id, err := arn.ResourceID(*a)
		if err != nil {
			return nil, err
		}

		appName, _ := c.split(&id)

		if appName == appID {
			arns = append(arns, a)
		}
	}

	return &ecs.ListServicesOutput{
		ServiceArns: arns,
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
