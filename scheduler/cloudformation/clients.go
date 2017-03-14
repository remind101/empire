package cloudformation

import (
	"time"

	awswaiter "github.com/aws/aws-sdk-go/private/waiter"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/pmylund/go-cache"
	"github.com/remind101/empire/pkg/arn"
)

var (
	defaultExpiration = 30 * time.Minute
	defaultPurge      = 30 * time.Second // Purge items every 30 seconds.
)

// cacher duck types the go-cache interface.
type cacher interface {
	Set(k string, x interface{}, d time.Duration)
	Get(k string) (interface{}, bool)
}

// cachingECSClient wraps an ecsClient to perform some performance
// optimizations, by taking advantage of the fact that task definitions are
// essentially immutable and can be cached forever.
type cachingECSClient struct {
	ecsClient

	// cache of task definitions
	taskDefinitions cacher
}

// ecsWithCaching wraps an ecs.ECS client with caching.
func ecsWithCaching(ecs *ECS) *cachingECSClient {
	return &cachingECSClient{
		ecsClient:       ecs,
		taskDefinitions: cache.New(defaultExpiration, defaultPurge),
	}
}

// DescribeTaskDefinition will use the task definition from cache if provided
// with a task definition ARN.
func (c *cachingECSClient) DescribeTaskDefinition(input *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	if _, err := arn.Parse(*input.TaskDefinition); err != nil {
		return c.ecsClient.DescribeTaskDefinition(input)
	}

	if v, ok := c.taskDefinitions.Get(*input.TaskDefinition); ok {
		return &ecs.DescribeTaskDefinitionOutput{
			TaskDefinition: v.(*ecs.TaskDefinition),
		}, nil
	}

	resp, err := c.ecsClient.DescribeTaskDefinition(input)
	if err != nil {
		return resp, err
	}

	c.taskDefinitions.Set(*resp.TaskDefinition.TaskDefinitionArn, resp.TaskDefinition, 0)

	return resp, err
}

// ECS augments the ecs.ECS client with extra waiters.
type ECS struct {
	*ecs.ECS
}

// WaitUntilTasksNotPending waits until all the given tasks are either RUNNING
// or STOPPED.
func (c *ECS) WaitUntilTasksNotPending(input *ecs.DescribeTasksInput) error {
	waiterCfg := awswaiter.Config{
		Operation:   "DescribeTasks",
		Delay:       6,
		MaxAttempts: 100,
		Acceptors: []awswaiter.WaitAcceptor{
			{
				State:    "failure",
				Matcher:  "pathAny",
				Argument: "failures[].reason",
				Expected: "MISSING",
			},
			{
				State:    "success",
				Matcher:  "pathAll",
				Argument: "tasks[].lastStatus",
				Expected: "RUNNING",
			},
			{
				State:    "success",
				Matcher:  "pathAll",
				Argument: "tasks[].lastStatus",
				Expected: "STOPPED",
			},
		},
	}

	w := awswaiter.Waiter{
		Client: c.ECS,
		Input:  input,
		Config: waiterCfg,
	}
	return w.Wait()
}
