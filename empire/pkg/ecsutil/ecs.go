package ecsutil

import (
	"fmt"
	"time"

	"github.com/awslabs/aws-sdk-go/aws"
	"github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/remind101/pkg/trace"
	"golang.org/x/net/context"
)

// ECS represents our ECS client interface.
type ECS interface {
	// Task Definitions
	RegisterTaskDefinition(context.Context, *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error)
	DescribeTaskDefinition(context.Context, *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error)

	// Services
	CreateService(context.Context, *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error)
	DeleteService(context.Context, *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error)
	UpdateService(context.Context, *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error)
	ListServices(context.Context, *ecs.ListServicesInput) (*ecs.ListServicesOutput, error)
	DescribeServices(context.Context, *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error)

	// Tasks
	ListTasks(context.Context, *ecs.ListTasksInput) (*ecs.ListTasksOutput, error)
	StopTask(context.Context, *ecs.StopTaskInput) (*ecs.StopTaskOutput, error)
	DescribeTasks(context.Context, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
}

// newECSClient builds a new ECS client with autopagination and tracing.
func newECSClient(config *aws.Config) ECS {
	ecs := ecs.New(config)
	return &autoPaginatedClient{
		ECS: &ecsClient{ECS: ecs},
	}
}

// ecsClient is a base ECS client implementation
type ecsClient struct {
	*ecs.ECS

	// a timer used to throttle the RegisterTaskDefinition calls. ECS only
	// allows 60/min. See http://docs.aws.amazon.com/AmazonECS/latest/developerguide/service_limits.html
	tdThrottle *time.Ticker
}

func (c *ecsClient) CreateService(ctx context.Context, input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.CreateService(input)
	done(err, "CreateService", "service-name", stringField(input.ServiceName), "desired-count", intField(input.DesiredCount), "task-definition", stringField(input.TaskDefinition))
	return resp, err
}

func (c *ecsClient) DeleteService(ctx context.Context, input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.DeleteService(input)
	done(err, "DeleteService", "service-name", stringField(input.Service))
	return resp, err
}

func (c *ecsClient) UpdateService(ctx context.Context, input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.UpdateService(input)
	done(err, "UpdateService", "service-name", stringField(input.Service), "desired-count", intField(input.DesiredCount), "task-definition", stringField(input.TaskDefinition))
	return resp, err
}

func (c *ecsClient) RegisterTaskDefinition(ctx context.Context, input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	fmt.Println("Here")
	if c.tdThrottle == nil {
		// Only allow 1 task definition per second.
		c.tdThrottle = time.NewTicker(time.Second)
	}

	<-c.tdThrottle.C

	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.RegisterTaskDefinition(input)
	done(err, "RegisterTaskDefinition", "family", stringField(input.Family))
	return resp, err
}

func (c *ecsClient) DescribeTaskDefinition(ctx context.Context, input *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.DescribeTaskDefinition(input)
	done(err, "DescribeTaskDefinition", "task-definition", stringField(input.TaskDefinition))
	return resp, err
}

func (c *ecsClient) ListServices(ctx context.Context, input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.ListServices(input)
	done(err, "ListServices")
	return resp, err
}

func (c *ecsClient) DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.DescribeServices(input)
	done(err, "DescribeServices", "services", len(input.Services))
	return resp, err
}

func (c *ecsClient) ListTasks(ctx context.Context, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.ListTasks(input)
	done(err, "ListTasks")
	return resp, err
}

func (c *ecsClient) DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.DescribeTasks(input)
	done(err, "DescribeTasks", "tasks", len(input.Tasks))
	return resp, err
}

func (c *ecsClient) StopTask(ctx context.Context, input *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.StopTask(input)
	done(err, "StopTask", "task", stringField(input.Task))
	return resp, err
}

func stringField(v *string) string {
	if v != nil {
		return *v
	}

	return "<nil>"
}

func intField(v *int64) string {
	if v != nil {
		return fmt.Sprintf("%d", *v)
	}

	return "<nil>"
}

// autoPaginatedClient is an ECS implementation that will autopaginate
// responses.
type autoPaginatedClient struct {
	ECS
}

func (c *autoPaginatedClient) ListServices(ctx context.Context, input *ecs.ListServicesInput) (*ecs.ListServicesOutput, error) {
	var (
		nextMarker *string
		arns       []*string
	)

	for {
		resp, err := c.ECS.ListServices(ctx, &ecs.ListServicesInput{
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

func (c *autoPaginatedClient) ListTasks(ctx context.Context, input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	var (
		nextMarker *string
		arns       []*string
	)

	for {
		resp, err := c.ECS.ListTasks(ctx, &ecs.ListTasksInput{
			Cluster:     input.Cluster,
			ServiceName: input.ServiceName,
			NextToken:   nextMarker,
		})
		if err != nil {
			return nil, err
		}

		arns = append(arns, resp.TaskARNs...)

		nextMarker = resp.NextToken
		if nextMarker == nil || *nextMarker == "" {
			// No more items
			break
		}
	}

	return &ecs.ListTasksOutput{
		TaskARNs: arns,
	}, nil
}

const describeServiceLimit = 10

// TODO: Parallelize this.
func (c *autoPaginatedClient) DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	var (
		arns     = input.Services
		max      = len(arns)
		services []*ecs.Service
	)

	// Slice off chunks of 10 arns.
	for i := 0; true; i += describeServiceLimit {
		// End point for this chunk.
		e := i + describeServiceLimit
		if e >= max {
			e = max
		}

		chunk := arns[i:e]

		resp, err := c.ECS.DescribeServices(ctx, &ecs.DescribeServicesInput{
			Cluster:  input.Cluster,
			Services: chunk,
		})
		if err != nil {
			return nil, err
		}

		services = append(services, resp.Services...)

		// No more chunks.
		if max == e {
			break
		}
	}

	return &ecs.DescribeServicesOutput{
		Services: services,
	}, nil
}
