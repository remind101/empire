package cloudformation

import (
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestECSServiceResource_Create(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs:     e,
		postfix: func() string { return "-A" },
	}

	e.On("CreateService", &ecs.CreateServiceInput{
		ServiceName:  aws.String("acme-inc-web-A"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(1),
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceArn: aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-A"),
		},
	}, nil)

	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-A")},
	}).Return(nil)

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
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-A", id)
	assert.Nil(t, data)

	e.AssertExpectations(t)
}

func TestECSServiceResource_Update(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs: e,
	}

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:        aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:        aws.String("cluster"),
		DesiredCount:   aws.Int64(2),
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
	}).Return(&ecs.UpdateServiceOutput{}, nil)

	id, data, err := p.Provision(Request{
		RequestType:        Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web",
		ResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("cluster"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   intValue(2),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("cluster"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   intValue(1),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:1"),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Nil(t, data)

	e.AssertExpectations(t)
}

func TestECSServiceResource_Update_RequiresReplacement(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs:     e,
		postfix: func() string { return "-B" },
	}

	e.On("CreateService", &ecs.CreateServiceInput{
		ServiceName:    aws.String("acme-inc-web-B"),
		Cluster:        aws.String("clusterB"),
		DesiredCount:   aws.Int64(2),
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceArn: aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-B"),
		},
	}, nil)

	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("clusterB"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-B")},
	}).Return(nil)

	id, data, err := p.Provision(Request{
		RequestType:        Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-A",
		ResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("clusterB"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   intValue(2),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("clusterA"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   intValue(1),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:1"),
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-B", id)
	assert.Nil(t, data)

	e.AssertExpectations(t)
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

	e.AssertExpectations(t)
}

func TestECSServiceResource_Delete_NotActive(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs: e,
	}

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:      aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(0),
	}).Return(&ecs.UpdateServiceOutput{}, awserr.New("ServiceNotActiveException", "Service was not ACTIVE", errors.New("")))

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

	e.AssertExpectations(t)
}

func TestRequiresReplacement(t *testing.T) {
	tests := []struct {
		new, old ECSServiceProperties
		out      bool
	}{
		{
			ECSServiceProperties{Cluster: aws.String("cluster"), TaskDefinition: aws.String("td:2"), DesiredCount: intValue(1)},
			ECSServiceProperties{Cluster: aws.String("cluster"), TaskDefinition: aws.String("td:1"), DesiredCount: intValue(0)},
			false,
		},

		{
			ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: intValue(8080), LoadBalancerName: aws.String("elb")}}},
			ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: intValue(8080), LoadBalancerName: aws.String("elb")}}},
			false,
		},

		// Can't change clusters.
		{
			ECSServiceProperties{Cluster: aws.String("clusterB")},
			ECSServiceProperties{Cluster: aws.String("clusterA")},
			true,
		},

		// Can't change name.
		{
			ECSServiceProperties{ServiceName: aws.String("acme-inc-B")},
			ECSServiceProperties{ServiceName: aws.String("acme-inc-A")},
			true,
		},

		// Can't change role.
		{
			ECSServiceProperties{Role: aws.String("roleB")},
			ECSServiceProperties{Role: aws.String("roleA")},
			true,
		},

		// Can't change load balancers
		{
			ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: intValue(8080), LoadBalancerName: aws.String("elbB")}}},
			ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: intValue(8080), LoadBalancerName: aws.String("elbA")}}},
			true,
		},
	}

	for _, tt := range tests {
		out := requiresReplacement(&tt.new, &tt.old)
		assert.Equal(t, tt.out, out)
	}
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

func (m *mockECS) WaitUntilServicesStable(input *ecs.DescribeServicesInput) error {
	args := m.Called(input)
	return args.Error(0)
}
