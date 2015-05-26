// Package ecsutil is a layer on top of Amazon ECS to provide an app aware ECS
// client.
package ecsutil

import (
	"strings"

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
	*ecs.ECS

	// The delimiter to use to separate app name from service type. Zero
	// value is the DefaultDelimiter.
	Delimiter string
}

// NewClient returns a new Client instance using the provided aws.Config.
func NewClient(config *aws.Config) *Client {
	ecs := ecs.New(config)
	return &Client{
		ECS: ecs,
	}
}

// CreateAppService creates a new ecs service for the app.
func (c *Client) CreateAppService(app string, input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	input.ServiceName = c.prefix(app, input.ServiceName)
	input.TaskDefinition = c.prefix(app, input.TaskDefinition)
	return c.ECS.CreateService(input)
}

// DeleteAppService deletes the service for the app.
func (c *Client) DeleteAppService(app string, input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	input.Service = c.prefix(app, input.Service)
	return c.ECS.DeleteService(input)
}

// UpdateAppService updates the service for the app.
func (c *Client) UpdateAppService(app string, input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	input.Service = c.prefix(app, input.Service)
	input.TaskDefinition = c.prefix(app, input.TaskDefinition)
	return c.ECS.UpdateService(input)
}

// RegisterAppTaskDefinition register a task definition for the app.
func (c *Client) RegisterAppTaskDefinition(app string, input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	input.Family = c.prefix(app, input.Family)
	return c.ECS.RegisterTaskDefinition(input)
}

// ListAppServices lists all services for the app.
func (c *Client) ListAppServices(app string, input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	resp, err := c.listServices(input)
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

func (c *Client) listServices(input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	var (
		nextMarker *string
		arns       []*string
	)

	for {
		resp, err := c.ECS.ListServices(&ecs.ListServicesInput{
			Cluster:   input.Cluster,
			NextToken: nextMarker,
		})
		if err != nil {
			return resp, err
		}

		arns = append(arns, resp.ServiceARNs...)

		nextMarker = resp.NextToken
		if nextMarker == nil || *nextMarker == "" {
			// No more items
			break
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
