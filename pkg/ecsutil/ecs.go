package ecsutil

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/pkg/trace"
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
	ListServicesPages(context.Context, *ecs.ListServicesInput, func(*ecs.ListServicesOutput, bool) bool) error
	DescribeServices(context.Context, *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error)

	// Tasks
	ListTasksPages(context.Context, *ecs.ListTasksInput, func(*ecs.ListTasksOutput, bool) bool) error
	StopTask(context.Context, *ecs.StopTaskInput) (*ecs.StopTaskOutput, error)
	DescribeTasks(context.Context, *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error)
	RunTask(context.Context, *ecs.RunTaskInput) (*ecs.RunTaskOutput, error)
}

// newECSClient builds a new ECS client with autopagination and tracing.
func newECSClient(p client.ConfigProvider) ECS {
	ecs := ecs.New(p)
	return &limitedClient{
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

func (c *ecsClient) ListServicesPages(ctx context.Context, input *ecs.ListServicesInput, fn func(*ecs.ListServicesOutput, bool) bool) error {
	ctx, done := trace.Trace(ctx)
	err := c.ECS.ListServicesPages(input, fn)
	done(err, "ListServicesPages")
	return err
}

func (c *ecsClient) DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.DescribeServices(input)
	done(err, "DescribeServices", "services", len(input.Services))
	return resp, err
}

func (c *ecsClient) ListTasksPages(ctx context.Context, input *ecs.ListTasksInput, fn func(*ecs.ListTasksOutput, bool) bool) error {
	ctx, done := trace.Trace(ctx)
	err := c.ECS.ListTasksPages(input, fn)
	done(err, "ListTasksPages")
	return err
}

func (c *ecsClient) DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	return c.describeTasks(ctx, input)
}

func (c *ecsClient) describeTasks(ctx context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
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

func (c *ecsClient) RunTask(ctx context.Context, input *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.RunTask(input)
	done(err, "RunTask", "taskDefinition", stringField(input.TaskDefinition))
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

// limitedClient is an ECS client that will handle limits in certain service
// calls.
type limitedClient struct {
	ECS
}

const describeServicesLimit = 10

// TODO: Parallelize this.
func (c *limitedClient) DescribeServices(ctx context.Context, input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	var (
		arns     = input.Services
		max      = len(arns)
		services []*ecs.Service
	)

	// Slice off chunks of 10 arns.
	for i := 0; true; i += describeServicesLimit {
		// End point for this chunk.
		e := i + describeServicesLimit
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

const describeTasksLimit = 100

func (c *limitedClient) DescribeTasks(ctx context.Context, input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	var (
		arns  = input.Tasks
		max   = len(arns)
		tasks []*ecs.Task
	)

	// Slice off chunks of 100 arns.
	for i := 0; true; i += describeTasksLimit {
		// End point for this chunk.
		e := i + describeTasksLimit
		if e >= max {
			e = max
		}

		chunk := arns[i:e]

		resp, err := c.ECS.DescribeTasks(ctx, &ecs.DescribeTasksInput{
			Cluster: input.Cluster,
			Tasks:   chunk,
		})
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, resp.Tasks...)

		// No more chunks.
		if max == e {
			break
		}
	}

	return &ecs.DescribeTasksOutput{
		Tasks: tasks,
	}, nil
}

func NewLogConfiguration(logDriver string, logOpts []string) *ecs.LogConfiguration {
	if logDriver == "" {
		// Default to the docker daemon default logging driver.
		return nil
	}

	logOptions := make(map[string]*string)

	for _, opt := range logOpts {
		logOpt := strings.SplitN(opt, "=", 2)
		if len(logOpt) == 2 {
			logOptions[logOpt[0]] = &logOpt[1]
		}
	}

	return &ecs.LogConfiguration{
		LogDriver: aws.String(logDriver),
		Options:   logOptions,
	}
}
