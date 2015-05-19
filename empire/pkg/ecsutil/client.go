// package ecsutil is a layer on top of Amazon ECS to provide an app aware ECS
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

// UpdateService updates the service for the app.
func (c *Client) UpdateAppService(app string, input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	input.Service = c.prefix(app, input.Service)
	input.TaskDefinition = c.prefix(app, input.TaskDefinition)
	return c.ECS.UpdateService(input)
}

// RegisterTaskDefinition register a task definition for the app.
func (c *Client) RegisterAppTaskDefinition(app string, input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	input.Family = c.prefix(app, input.Family)
	return c.ECS.RegisterTaskDefinition(input)
}

// ListAppServices lists all services for an app.
func (c *Client) ListAppServices(app string, input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	return c.listFilterServices(input, func(ARN string) (bool, error) {
		id, err := arn.ResourceID(ARN)
		if err != nil {
			return false, err
		}

		appName, _ := c.split(&id)

		if appName == app {
			return true, nil
		}
		return false, nil
	})
}

// ListAppService lists a service for an app process.
func (c *Client) ListAppService(app string, process string, input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	return c.listFilterServices(input, func(ARN string) (bool, error) {
		id, err := arn.ResourceID(ARN)
		if err != nil {
			return false, err
		}

		appName, procType := c.split(&id)

		if appName == app && *procType == process {
			return true, nil
		}
		return false, nil
	})
}

// listFilterServices applies a filter function to service ARNS, returning only
// those services for which filterFn returns true.
func (c *Client) listFilterServices(input *ecs.ListServicesInput, filterFn func(string) (bool, error)) (*ecs.ListServicesOutput, error) {
	resp, err := c.ECS.ListServices(input)
	if err != nil {
		return resp, err
	}

	var arns []*string
	for _, a := range resp.ServiceARNs {
		if a == nil {
			continue
		}

		ok, err := filterFn(*a)
		if err != nil {
			return resp, err
		}
		if ok {
			arns = append(arns, a)
		}
	}

	resp.ServiceARNs = arns
	return resp, nil
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
