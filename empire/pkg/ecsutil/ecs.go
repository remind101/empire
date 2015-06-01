package ecsutil

import (
	"fmt"

	"github.com/awslabs/aws-sdk-go/service/ecs"
	"github.com/remind101/pkg/trace"
	"golang.org/x/net/context"
)

// ECS is an ECS client with tracing.
type ECS struct {
	*ecs.ECS
}

func (c *ECS) CreateService(ctx context.Context, input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.CreateService(input)
	done(err, "CreateService", "service-name", stringField(input.ServiceName), "desired-count", intField(input.DesiredCount), "task-definition", stringField(input.TaskDefinition))
	return resp, err
}

func (c *ECS) DeleteService(ctx context.Context, input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.DeleteService(input)
	done(err, "DeleteService", "service-name", stringField(input.Service))
	return resp, err
}

func (c *ECS) UpdateService(ctx context.Context, input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.UpdateService(input)
	done(err, "UpdateService", "service-name", stringField(input.Service), "desired-count", intField(input.DesiredCount), "task-definition", stringField(input.TaskDefinition))
	return resp, err
}

func (c *ECS) RegisterTaskDefinition(ctx context.Context, input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	ctx, done := trace.Trace(ctx)
	resp, err := c.ECS.RegisterTaskDefinition(input)
	done(err, "RegisterTaskDefinition", "family", stringField(input.Family))
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
