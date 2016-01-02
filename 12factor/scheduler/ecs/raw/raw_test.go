package raw

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/12factor"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStackBuilder_Build(t *testing.T) {
	c := new(mockECSClient)
	b := &StackBuilder{
		Cluster: "cluster",
		ecs:     c,
	}

	manifest := twelvefactor.Manifest{
		App: twelvefactor.App{
			Name: "app",
			ID:   "app",
			Env: map[string]string{
				"RAILS_ENV": "production",
			},
		},
		Processes: []twelvefactor.Process{
			{
				Name:      "web",
				CPUShares: 256,
				Memory:    int(1 * bytesize.GB),
			},
		},
	}

	c.On("RegisterTaskDefinition", &ecs.RegisterTaskDefinitionInput{
		Family: aws.String("app--web"),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Name:      aws.String("web"),
				Cpu:       aws.Int64(256),
				Memory:    aws.Int64(1024),
				Image:     aws.String(""),
				Essential: aws.Bool(true),
				Environment: []*ecs.KeyValuePair{
					{
						Name:  aws.String("RAILS_ENV"),
						Value: aws.String("production"),
					},
				},
			},
		},
	}).Return(&ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:1"),
		},
	}, nil)
	c.On("CreateService", &ecs.CreateServiceInput{
		Cluster:        aws.String("cluster"),
		DesiredCount:   aws.Int64(0),
		Role:           aws.String(""),
		ServiceName:    aws.String("app--web"),
		TaskDefinition: aws.String("arn:aws:ecs:us-west-2:012345678910:task-definition/app--web:1"),
	}).Return(&ecs.CreateServiceOutput{}, nil)
	err := b.Build(manifest)
	assert.NoError(t, err)
}

func TestStackBuilder_Remove(t *testing.T) {
	c := new(mockECSClient)
	b := &StackBuilder{
		Cluster: "cluster",
		ecs:     c,
	}

	c.On("ListServicesPages", &ecs.ListServicesInput{
		Cluster: aws.String("cluster"),
	}).Return(nil, []*ecs.ListServicesOutput{
		{
			ServiceArns: []*string{
				aws.String("arn:aws:ecs:us-east-1:012345678910:service/app--web"),
			},
		},
	})
	c.On("UpdateService", &ecs.UpdateServiceInput{
		Cluster:      aws.String("cluster"),
		Service:      aws.String("app--web"),
		DesiredCount: aws.Int64(0),
	}).Return(&ecs.UpdateServiceOutput{}, nil)
	c.On("DeleteService", &ecs.DeleteServiceInput{
		Cluster: aws.String("cluster"),
		Service: aws.String("app--web"),
	}).Return(&ecs.DeleteServiceOutput{}, nil)
	err := b.Remove("app")
	assert.NoError(t, err)
}

func TestStackBuilder_Services(t *testing.T) {
	c := new(mockECSClient)
	b := &StackBuilder{
		Cluster: "cluster",
		ecs:     c,
	}

	c.On("ListServicesPages", &ecs.ListServicesInput{
		Cluster: aws.String("cluster"),
	}).Return(nil, []*ecs.ListServicesOutput{
		{
			ServiceArns: []*string{
				aws.String("arn:aws:ecs:us-east-1:012345678910:service/app--web"),
			},
		},
	})
	services, err := b.Services("app")
	assert.NoError(t, err)
	assert.Equal(t, services, map[string]string{
		"web": "app--web",
	})
}

func TestStackBuilder_Services_Pagination(t *testing.T) {
	c := new(mockECSClient)
	b := &StackBuilder{
		Cluster: "cluster",
		ecs:     c,
	}

	c.On("ListServicesPages", &ecs.ListServicesInput{
		Cluster: aws.String("cluster"),
	}).Return(nil, []*ecs.ListServicesOutput{
		{
			ServiceArns: []*string{
				aws.String("arn:aws:ecs:us-east-1:012345678910:service/app--web"),
			},
		},
		{
			ServiceArns: []*string{
				aws.String("arn:aws:ecs:us-east-1:012345678910:service/app--worker"),
			},
		},
	})
	services, err := b.Services("app")
	assert.NoError(t, err)
	assert.Equal(t, services, map[string]string{
		"web":    "app--web",
		"worker": "app--worker",
	})
}

func TestStackBuilder_Services_Dirty(t *testing.T) {
	c := new(mockECSClient)
	b := &StackBuilder{
		Cluster: "cluster",
		ecs:     c,
	}

	c.On("ListServicesPages", &ecs.ListServicesInput{
		Cluster: aws.String("cluster"),
	}).Return(nil, []*ecs.ListServicesOutput{
		{
			ServiceArns: []*string{
				aws.String("arn:aws:ecs:us-east-1:012345678910:service/app"),
				aws.String("arn:aws:ecs:us-east-1:012345678910:service/app--web"),
				nil,
			},
		},
	})
	services, err := b.Services("app")
	assert.NoError(t, err)
	assert.Equal(t, services, map[string]string{
		"web": "app--web",
	})
}

// mockECSClient is an implementation of the ecsClient interface for testing.
type mockECSClient struct {
	mock.Mock
}

func (c *mockECSClient) ListServicesPages(input *ecs.ListServicesInput, fn func(*ecs.ListServicesOutput, bool) bool) error {
	args := c.Called(input)
	for _, resp := range args.Get(1).([]*ecs.ListServicesOutput) {
		if !fn(resp, false) {
			break
		}
	}
	return args.Error(0)
}

func (c *mockECSClient) DeleteService(input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.DeleteServiceOutput), args.Error(1)
}

func (c *mockECSClient) RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.RegisterTaskDefinitionOutput), args.Error(1)
}

func (c *mockECSClient) CreateService(input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.CreateServiceOutput), args.Error(1)
}

func (c *mockECSClient) UpdateService(input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*ecs.UpdateServiceOutput), args.Error(1)
}
