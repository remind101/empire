package cloudformation

import (
	"errors"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
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
		ClientToken:  aws.String("dxRU5tYsnzt"),
		ServiceName:  aws.String("acme-inc-web-dxRU5tYsnzt"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(1),
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt"),
			Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")}},
		},
	}, nil)

	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt")},
	}).Return(nil)

	id, data, err := p.Provision(ctx, Request{
		StackId:     "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:   "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType: Create,
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: intValue(1),
		},
		OldResourceProperties: &ECSServiceProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt", id)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New"})

	e.AssertExpectations(t)
}

func TestECSServiceResource_Create_Canceled(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs: e,
	}

	e.On("CreateService", &ecs.CreateServiceInput{
		ClientToken:  aws.String("dxRU5tYsnzt"),
		ServiceName:  aws.String("acme-inc-web-dxRU5tYsnzt"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(1),
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt"),
			Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")}},
		},
	}, nil)

	ctx, cancel := context.WithCancel(ctx)
	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt")},
	}).Return(nil).Run(func(mock.Arguments) {
		cancel()
		time.Sleep(1 * time.Second)
	})

	_, data, err := p.Provision(ctx, Request{
		StackId:     "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:   "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType: Create,
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: intValue(1),
		},
		OldResourceProperties: &ECSServiceProperties{},
	})
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New"})

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
	}).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				Deployments: []*ecs.Deployment{
					&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")},
					&ecs.Deployment{Id: aws.String("Old"), Status: aws.String("ACTIVE")},
				},
			},
		},
		nil,
	)

	id, data, err := p.Provision(ctx, Request{
		StackId:            "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:          "411f3f38-565f-4216-a711-aeafd5ba635e",
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
	assert.Equal(t, data, map[string]string{"DeploymentId": "New"})

	e.AssertExpectations(t)
}

func TestECSServiceResource_Update_RequiresReplacement(t *testing.T) {
	e := new(mockECS)
	p := &ECSServiceResource{
		ecs: e,
	}

	e.On("CreateService", &ecs.CreateServiceInput{
		ClientToken:    aws.String("dxRU5tYsnzt"),
		ServiceName:    aws.String("acme-inc-web-dxRU5tYsnzt"),
		Cluster:        aws.String("clusterB"),
		DesiredCount:   aws.Int64(2),
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt"),
			Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")}},
		},
	}, nil)

	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("clusterB"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt")},
	}).Return(nil)

	id, data, err := p.Provision(ctx, Request{
		StackId:            "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:          "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType:        Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt",
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
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt", id)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New"})

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
	}).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				Deployments: []*ecs.Deployment{
					&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")},
					&ecs.Deployment{Id: aws.String("Old"), Status: aws.String("ACTIVE")},
				},
			},
		},
		nil,
	)

	e.On("DeleteService", &ecs.DeleteServiceInput{
		Service: aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster: aws.String("cluster"),
	}).Return(&ecs.DeleteServiceOutput{}, nil)

	id, data, err := p.Provision(ctx, Request{
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

	id, data, err := p.Provision(ctx, Request{
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
