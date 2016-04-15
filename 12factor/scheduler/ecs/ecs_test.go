package ecs

import (
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/12factor"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Ensure that we implement the Scheduler interface.
var _ twelvefactor.Scheduler = (*Scheduler)(nil)

func TestScheduler_Up(t *testing.T) {
	b := new(mockStackBuilder)
	s := &Scheduler{
		StackBuilder: b,
	}

	manifest := twelvefactor.Manifest{}
	b.On("Build", manifest).Return(nil)
	err := s.Up(manifest)
	assert.NoError(t, err)
}

func TestScheduler_Remove(t *testing.T) {
	b := new(mockStackBuilder)
	s := &Scheduler{
		StackBuilder: b,
	}

	b.On("Remove", "app").Return(nil)
	err := s.Remove("app")
	assert.NoError(t, err)
}

func TestScheduler_ScaleProcess(t *testing.T) {
	b := new(mockStackBuilder)
	c := new(mockECSClient)
	s := &Scheduler{
		Cluster:      "cluster",
		StackBuilder: b,
		ecs:          c,
	}

	b.On("Services", "app").Return(map[string]string{
		"web": "app--web",
	}, nil)
	c.On("UpdateService", &ecs.UpdateServiceInput{
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(1),
		Service:      aws.String("app--web"),
	}).Return(&ecs.UpdateServiceOutput{}, nil)
	err := s.ScaleProcess("app", "web", 1)
	assert.NoError(t, err)
}

func TestScheduler_ScaleProcess_NotFound(t *testing.T) {
	b := new(mockStackBuilder)
	c := new(mockECSClient)
	s := &Scheduler{
		Cluster:      "cluster",
		StackBuilder: b,
		ecs:          c,
	}

	b.On("Services", "app").Return(map[string]string{}, nil)
	err := s.ScaleProcess("app", "web", 1)
	assert.Error(t, err, "web process not found")
}

func TestScheduler_Tasks(t *testing.T) {
	b := new(mockStackBuilder)
	c := new(mockECSClient)
	s := &Scheduler{
		Cluster:      "cluster",
		StackBuilder: b,
		ecs:          c,
	}

	b.On("Services", "app").Return(map[string]string{
		"web": "app--web",
	}, nil)
	c.On("ListTasks", &ecs.ListTasksInput{
		Cluster:     aws.String("cluster"),
		ServiceName: aws.String("app--web"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
		},
	}, nil)
	c.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("app--web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{TaskDefinition: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:1")},
		},
	}, nil)
	c.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:1"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name:    aws.String("web"),
					Command: []*string{aws.String("./bin/web")},
					Memory:  aws.Int64(5),
					Cpu:     aws.Int64(256),
				},
			},
		},
	}, nil)
	c.On("DescribeTasks", &ecs.DescribeTasksInput{
		Cluster: aws.String("cluster"),
		Tasks: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
		},
	}).Return(&ecs.DescribeTasksOutput{
		Tasks: []*ecs.Task{
			{
				TaskArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
				LastStatus: aws.String("RUNNING"),
				StartedAt:  aws.Time(time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)),
			},
		},
	}, nil)
	tasks, err := s.Tasks("app")
	assert.NoError(t, err)
	assert.Equal(t, tasks, []twelvefactor.Task{
		{
			ID:        "0b69d5c0-d655-4695-98cd-5d2d526d9d5a",
			Version:   "TODO",
			Process:   "web",
			State:     "RUNNING",
			UpdatedAt: time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC),
			Command:   []string{"./bin/web"},
			Memory:    5242880,
			CPUShares: 256,
		},
	})
}

func TestScheduler_RestartProcess(t *testing.T) {
	b := new(mockStackBuilder)
	c := new(mockECSClient)
	s := &Scheduler{
		Cluster:      "cluster",
		StackBuilder: b,
		ecs:          c,
	}

	b.On("Services", "app").Return(map[string]string{"web": "app--web", "worker": "app--worker"}, nil)
	c.On("DescribeServices", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("app--web")},
	}).Return(&ecs.DescribeServicesOutput{
		Services: []*ecs.Service{
			{TaskDefinition: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:1")},
		},
	}, nil)
	c.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:1"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			Family: aws.String("app--web"),
		},
	}, nil)
	c.On("RegisterTaskDefinition", &ecs.RegisterTaskDefinitionInput{
		Family: aws.String("app--web"),
	}).Return(&ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:2"),
		},
	}, nil)
	c.On("UpdateService", &ecs.UpdateServiceInput{
		Cluster:        aws.String("cluster"),
		TaskDefinition: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:2"),
		Service:        aws.String("app--web"),
	}).Return(&ecs.UpdateServiceOutput{}, nil)

	err := s.RestartProcess("app", "web")
	assert.NoError(t, err)

	b.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestScheduler_StopTask(t *testing.T) {
	b := new(mockStackBuilder)
	c := new(mockECSClient)
	s := &Scheduler{
		Cluster:      "cluster",
		StackBuilder: b,
		ecs:          c,
	}

	c.On("StopTask", &ecs.StopTaskInput{
		Cluster: aws.String("cluster"),
		Task:    aws.String("uuid"),
	}).Return(&ecs.StopTaskOutput{}, nil)

	err := s.StopTask("uuid")
	assert.NoError(t, err)

	b.AssertExpectations(t)
	c.AssertExpectations(t)
}

// mockECSClient is an implementation of the ecsClient interface for testing.
type mockECSClient struct {
	mock.Mock
}

func (c *mockECSClient) DescribeServices(input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.DescribeServicesOutput), args.Error(1)
}

func (c *mockECSClient) UpdateService(input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.UpdateServiceOutput), args.Error(1)
}

func (c *mockECSClient) ListTasks(input *ecs.ListTasksInput) (*ecs.ListTasksOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.ListTasksOutput), args.Error(1)
}

func (c *mockECSClient) DescribeTasks(input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.DescribeTasksOutput), args.Error(1)
}

func (c *mockECSClient) StopTask(input *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.StopTaskOutput), args.Error(1)
}

func (c *mockECSClient) DescribeTaskDefinition(input *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.DescribeTaskDefinitionOutput), args.Error(1)
}

func (c *mockECSClient) RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.RegisterTaskDefinitionOutput), args.Error(1)
}

// mockStackBuilder is an implementation of the StackBuilder interface for
// testing.
type mockStackBuilder struct {
	mock.Mock
}

func (b *mockStackBuilder) Build(manifest twelvefactor.Manifest) error {
	args := b.Called(manifest)
	return args.Error(0)
}

func (b *mockStackBuilder) Remove(app string) error {
	args := b.Called(app)
	return args.Error(0)
}

func (b *mockStackBuilder) Services(app string) (map[string]string, error) {
	args := b.Called(app)
	return args.Get(0).(map[string]string), args.Error(1)
}
