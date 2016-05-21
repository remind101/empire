package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestECSServiceResource_Create(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs: e,
	}

	e.On("CreateService", &ecs.CreateServiceInput{
		ServiceName:  aws.String("acme-inc-web"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(1),
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceArn: aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		},
	}, nil)

	id, data, err := p.Provision(Request{
		RequestType: Create,
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: intValue(1),
		},
		OldResourceProperties: &ECSServiceProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Nil(t, data)
}

func TestECSServiceResource_Update(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs: e,
	}

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:      aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(2),
	}).Return(&ecs.UpdateServiceOutput{}, nil)

	id, data, err := p.Provision(Request{
		RequestType:        Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web",
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: intValue(2),
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: intValue(1),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Nil(t, data)
}

func TestECSServiceResource_Delete(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs: e,
	}

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:      aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(0),
	}).Return(&ecs.UpdateServiceOutput{}, nil)

	e.On("DeleteService", &ecs.DeleteServiceInput{
		Service: aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster: aws.String("cluster"),
	}).Return(&ecs.DeleteServiceOutput{}, nil)

	id, data, err := p.Provision(Request{
		RequestType:        Delete,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web",
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: intValue(1),
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: intValue(1),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Nil(t, data)
}

type mockECS struct {
	ecsClient
	mock.Mock
}

func (m *mockECS) CreateService(input *ecs.CreateServiceInput) (*ecs.CreateServiceOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.CreateServiceOutput), args.Error(1)
}

func (m *mockECS) UpdateService(input *ecs.UpdateServiceInput) (*ecs.UpdateServiceOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.UpdateServiceOutput), args.Error(1)
}

func (m *mockECS) DeleteService(input *ecs.DeleteServiceInput) (*ecs.DeleteServiceOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DeleteServiceOutput), args.Error(1)
}
