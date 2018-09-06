package ecs

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
)

func TestCachingECSClient_DescribeTaskDefiniton_WithArn(t *testing.T) {
	cache := newMockCacher()
	e := new(mockECSClient)
	c := &cachingECSClient{
		ecsClient:       e,
		taskDefinitions: cache,
	}

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10"),
		},
	}, nil).Once()

	resp, err := c.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10", *resp.TaskDefinition.TaskDefinitionArn)

	resp, err = c.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10", *resp.TaskDefinition.TaskDefinitionArn)
}

func TestCachingECSClient_DescribeTaskDefiniton_WithoutArn(t *testing.T) {
	e := new(mockECSClient)
	c := &cachingECSClient{
		ecsClient: e,
	}

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("hello_world"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10"),
		},
	}, nil).Twice()

	resp, err := c.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("hello_world"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10", *resp.TaskDefinition.TaskDefinitionArn)

	resp, err = c.DescribeTaskDefinition(&ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("hello_world"),
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678910:task-definition/hello_world:10", *resp.TaskDefinition.TaskDefinitionArn)
}

type mockCacher struct {
	m map[string]interface{}
}

func newMockCacher() *mockCacher {
	return &mockCacher{make(map[string]interface{})}
}

func (m *mockCacher) Set(k string, x interface{}, d time.Duration) {
	m.m[k] = x
}

func (m *mockCacher) Get(k string) (interface{}, bool) {
	x, ok := m.m[k]
	return x, ok
}
