package cloudformation

import (
	"errors"
	"net/http"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/request"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/remind101/empire/pkg/cloudformation/customresources"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRetryer(t *testing.T) {
	r := newRetryer()

	var min, max, total time.Duration
	for i := 0; i < r.NumMaxRetries; i++ {
		retryCount := i
		x := i
		if x > 8 {
			x = 8
		}

		min += time.Duration((1<<uint(x))*500) * time.Millisecond
		max += time.Duration((1<<uint(x))*1000) * time.Millisecond

		delay := r.RetryRules(&request.Request{
			RetryCount: retryCount,
			HTTPResponse: &http.Response{
				StatusCode: 503,
			},
		})
		total += delay
		t.Logf("delay(%d): %v", i, delay)
	}
	t.Logf("total(min): %v", min)
	t.Logf("total(max): %v", max)
	t.Logf("total(real): %v", total)

	assert.True(t, total > min)
	assert.True(t, total < max)
}

func TestECSServiceResource_Create(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("CreateService", &ecs.CreateServiceInput{
		ClientToken:  aws.String("dxRU5tYsnzt"),
		ServiceName:  aws.String("acme-inc-web-dxRU5tYsnzt"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(1),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceName: aws.String("acme-inc-web-dxRU5tYsnzt"),
			ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt"),
			Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")}},
		},
	}, nil)

	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("cluster"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt")},
	}).Return(nil)

	id, data, err := p.Provision(ctx, customresources.Request{
		StackId:     "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:   "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType: customresources.Create,
		ResourceProperties: &ECSServiceProperties{
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: customresources.Int(1),
		},
		OldResourceProperties: &ECSServiceProperties{},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt", id)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New", "Name": "acme-inc-web-dxRU5tYsnzt"})

	e.AssertExpectations(t)
}

func TestECSServiceResource_Create_Canceled(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("CreateService", &ecs.CreateServiceInput{
		ClientToken:  aws.String("dxRU5tYsnzt"),
		ServiceName:  aws.String("acme-inc-web-dxRU5tYsnzt"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(1),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceName: aws.String("acme-inc-web-dxRU5tYsnzt"),
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

	_, data, err := p.Provision(ctx, customresources.Request{
		StackId:     "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:   "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType: customresources.Create,
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: customresources.Int(1),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
		OldResourceProperties: &ECSServiceProperties{},
	})
	assert.Equal(t, context.Canceled, err)
	assert.Equal(t, map[string]string{"DeploymentId": "New", "Name": "acme-inc-web-dxRU5tYsnzt"}, data)

	e.AssertExpectations(t)
}

func TestECSServiceResource_Update(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:        aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:        aws.String("cluster"),
		DesiredCount:   aws.Int64(2),
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
	}).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				ServiceName: aws.String("acme-inc-web"),
				Deployments: []*ecs.Deployment{
					&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")},
					&ecs.Deployment{Id: aws.String("Old"), Status: aws.String("ACTIVE")},
				},
			},
		},
		nil,
	)

	id, data, err := p.Provision(ctx, customresources.Request{
		StackId:            "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:          "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType:        customresources.Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web",
		ResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("cluster"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(2),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("cluster"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(1),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:1"),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New", "Name": "acme-inc-web"})

	e.AssertExpectations(t)
}

func TestECSServiceResource_Update_SameDesiredCount(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:        aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:        aws.String("cluster"),
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
	}).Return(
		&ecs.UpdateServiceOutput{
			Service: &ecs.Service{
				ServiceName: aws.String("acme-inc-web"),
				Deployments: []*ecs.Deployment{
					&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")},
					&ecs.Deployment{Id: aws.String("Old"), Status: aws.String("ACTIVE")},
				},
			},
		},
		nil,
	)

	id, data, err := p.Provision(ctx, customresources.Request{
		StackId:            "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:          "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType:        customresources.Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web",
		ResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("cluster"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(2),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("cluster"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(2),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:1"),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New", "Name": "acme-inc-web"})

	e.AssertExpectations(t)
}

func TestECSServiceResource_Update_RequiresReplacement(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("CreateService", &ecs.CreateServiceInput{
		ClientToken:    aws.String("dxRU5tYsnzt"),
		ServiceName:    aws.String("acme-inc-web-dxRU5tYsnzt"),
		Cluster:        aws.String("clusterB"),
		DesiredCount:   aws.Int64(2),
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceName: aws.String("acme-inc-web-dxRU5tYsnzt"),
			ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt"),
			Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")}},
		},
	}, nil)

	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("clusterB"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt")},
	}).Return(nil)

	id, data, err := p.Provision(ctx, customresources.Request{
		StackId:            "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:          "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType:        customresources.Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt",
		ResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("clusterB"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(2),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("clusterA"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(1),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:1"),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt", id)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New", "Name": "acme-inc-web-dxRU5tYsnzt"})

	e.AssertExpectations(t)
}

func TestECSServiceResource_Update_Placement(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("CreateService", &ecs.CreateServiceInput{
		ClientToken:    aws.String("dxRU5tYsnzt"),
		ServiceName:    aws.String("acme-inc-web-dxRU5tYsnzt"),
		Cluster:        aws.String("clusterA"),
		DesiredCount:   aws.Int64(2),
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
		PlacementConstraints: []*ecs.PlacementConstraint{
			{Type: aws.String("memberOf"), Expression: aws.String("attribute:ecs.instance-type =~ t2.*")},
		},
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
	}).Return(&ecs.CreateServiceOutput{
		Service: &ecs.Service{
			ServiceName: aws.String("acme-inc-web-dxRU5tYsnzt"),
			ServiceArn:  aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt"),
			Deployments: []*ecs.Deployment{&ecs.Deployment{Id: aws.String("New"), Status: aws.String("PRIMARY")}},
		},
	}, nil)

	e.On("WaitUntilServicesStable", &ecs.DescribeServicesInput{
		Cluster:  aws.String("clusterA"),
		Services: []*string{aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt")},
	}).Return(nil)

	id, data, err := p.Provision(ctx, customresources.Request{
		StackId:            "arn:aws:cloudformation:us-east-1:012345678901:stack/acme-inc/bc66fd60-32be-11e6-902b-50d501eb4c17",
		RequestId:          "411f3f38-565f-4216-a711-aeafd5ba635e",
		RequestType:        customresources.Update,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt",
		ResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("clusterA"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(2),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:2"),
			PlacementConstraints: []ECSPlacementConstraint{
				{Type: aws.String("memberOf"), Expression: aws.String("attribute:ecs.instance-type =~ t2.*")},
			},
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:        aws.String("clusterA"),
			ServiceName:    aws.String("acme-inc-web"),
			DesiredCount:   customresources.Int(1),
			TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc:1"),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web-dxRU5tYsnzt", id)
	assert.Equal(t, data, map[string]string{"DeploymentId": "New", "Name": "acme-inc-web-dxRU5tYsnzt"})

	e.AssertExpectations(t)
}

func TestECSServiceResource_Delete(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:      aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(0),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
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

	id, data, err := p.Provision(ctx, customresources.Request{
		RequestType:        customresources.Delete,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web",
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: customresources.Int(1),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: customresources.Int(1),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Nil(t, data)

	e.AssertExpectations(t)
}

func TestECSServiceResource_Delete_NotActive(t *testing.T) {
	e := new(mockECS)
	p := newECSServiceProvisioner(&ECSServiceResource{
		ecs: e,
	})

	e.On("UpdateService", &ecs.UpdateServiceInput{
		Service:      aws.String("arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web"),
		Cluster:      aws.String("cluster"),
		DesiredCount: aws.Int64(0),
		DeploymentConfiguration: &ecs.DeploymentConfiguration{
			MaximumPercent: aws.Int64(100),
			MinimumHealthyPercent: aws.Int64(80),
		},
	}).Return(&ecs.UpdateServiceOutput{}, awserr.New("ServiceNotActiveException", "Service was not ACTIVE", errors.New("")))

	id, data, err := p.Provision(ctx, customresources.Request{
		RequestType:        customresources.Delete,
		PhysicalResourceId: "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web",
		ResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: customresources.Int(1),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
		OldResourceProperties: &ECSServiceProperties{
			Cluster:      aws.String("cluster"),
			ServiceName:  aws.String("acme-inc-web"),
			DesiredCount: customresources.Int(1),
			DeploymentConfiguration: &DeploymentConfiguration{
				MaximumPercent: customresources.Int(100),
				MinimumHealthyPercent: customresources.Int(80),
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:service/acme-inc-web", id)
	assert.Nil(t, data)

	e.AssertExpectations(t)
}

func TestECSTaskDefinition_Create(t *testing.T) {
	e := new(mockECS)
	s := new(mockEnvironmentStore)
	p := newECSTaskDefinitionProvisioner(&ECSTaskDefinitionResource{
		ecs:              e,
		environmentStore: s,
	})

	s.On("fetch", "003483d3-74b8-465d-8c2e-06e005dda776").Return([]*ecs.KeyValuePair{
		{
			Name:  aws.String("FOO"),
			Value: aws.String("bar"),
		},
	}, nil)

	s.On("fetch", "ccc8a1ac-32f9-4576-bec6-4ca36520deb3").Return([]*ecs.KeyValuePair{
		{
			Name:  aws.String("BAR"),
			Value: aws.String("foo"),
		},
	}, nil)

	e.On("RegisterTaskDefinition", &ecs.RegisterTaskDefinitionInput{
		Family: aws.String("acme-inc-web-f3ASgQEwwCZ"),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Environment: []*ecs.KeyValuePair{
					{
						Name:  aws.String("FOO"),
						Value: aws.String("bar"),
					},
					{
						Name:  aws.String("BAR"),
						Value: aws.String("foo"),
					},
				},
			},
		},
		PlacementConstraints: []*ecs.TaskDefinitionPlacementConstraint{
			{
				Type:       aws.String("memberOf"),
				Expression: aws.String("attribute:instance-type =~ t1.micro"),
			},
		},
	}).Return(&ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678901:task-definition/acme-inc-web"),
		},
	}, nil)

	id, data, err := p.Provision(ctx, customresources.Request{
		RequestType: customresources.Create,
		ResourceProperties: &ECSTaskDefinitionProperties{
			Family: aws.String("acme-inc-web"),
			ContainerDefinitions: []ContainerDefinition{
				{
					Environment: []string{
						"003483d3-74b8-465d-8c2e-06e005dda776",
						"ccc8a1ac-32f9-4576-bec6-4ca36520deb3",
					},
				},
			},
			PlacementConstraints: []ECSPlacementConstraint{
				{
					Type:       aws.String("memberOf"),
					Expression: aws.String("attribute:instance-type =~ t1.micro"),
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:task-definition/acme-inc-web", id)
	assert.Nil(t, data)

	e.AssertExpectations(t)
	s.AssertExpectations(t)
}

func TestECSTaskDefinition_Update(t *testing.T) {
	e := new(mockECS)
	s := new(mockEnvironmentStore)
	p := newECSTaskDefinitionProvisioner(&ECSTaskDefinitionResource{
		ecs:              e,
		environmentStore: s,
	})

	s.On("fetch", "003483d3-74b8-465d-8c2e-06e005dda776").Return([]*ecs.KeyValuePair{
		{
			Name:  aws.String("FOO"),
			Value: aws.String("bar"),
		},
	}, nil)

	s.On("fetch", "4f3f884b-8337-4847-9b81-141b5e322559").Return([]*ecs.KeyValuePair{
		{
			Name:  aws.String("BAR"),
			Value: aws.String("foo"),
		},
	}, nil)

	e.On("RegisterTaskDefinition", &ecs.RegisterTaskDefinitionInput{
		Family: aws.String("acme-inc-web-f3ASgQEwwCZ"),
		ContainerDefinitions: []*ecs.ContainerDefinition{
			{
				Environment: []*ecs.KeyValuePair{
					{
						Name:  aws.String("FOO"),
						Value: aws.String("bar"),
					},
					{
						Name:  aws.String("BAR"),
						Value: aws.String("foo"),
					},
				},
			},
		},
	}).Return(&ecs.RegisterTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			TaskDefinitionArn: aws.String("arn:aws:ecs:us-east-1:012345678901:task-definition/acme-inc-web"),
		},
	}, nil)

	id, data, err := p.Provision(ctx, customresources.Request{
		RequestType: customresources.Update,
		ResourceProperties: &ECSTaskDefinitionProperties{
			Family: aws.String("acme-inc-web"),
			ContainerDefinitions: []ContainerDefinition{
				{
					Environment: []string{
						"003483d3-74b8-465d-8c2e-06e005dda776",
						"4f3f884b-8337-4847-9b81-141b5e322559",
					},
				},
			},
		},
		OldResourceProperties: &ECSTaskDefinitionProperties{
			Family: aws.String("acme-inc-web"),
			ContainerDefinitions: []ContainerDefinition{
				{
					Environment: []string{
						"003483d3-74b8-465d-8c2e-06e005dda776",
						"ccc8a1ac-32f9-4576-bec6-4ca36520deb3",
					},
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "arn:aws:ecs:us-east-1:012345678901:task-definition/acme-inc-web", id)
	assert.Nil(t, data)

	e.AssertExpectations(t)
	s.AssertExpectations(t)
}

func TestECSEnvironment_Create(t *testing.T) {
	s := new(mockEnvironmentStore)
	p := newECSEnvironmentProvisioner(&ECSEnvironmentResource{
		environmentStore: s,
	})

	s.On("store", []*ecs.KeyValuePair{
		{
			Name:  aws.String("FOO"),
			Value: aws.String("bar"),
		},
	}).Return("56152438-5fef-4c96-bbe1-9cf92022ae75", nil)

	id, data, err := p.Provision(ctx, customresources.Request{
		RequestType: customresources.Create,
		ResourceProperties: &ECSEnvironmentProperties{
			Environment: []*ecs.KeyValuePair{
				{
					Name:  aws.String("FOO"),
					Value: aws.String("bar"),
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "56152438-5fef-4c96-bbe1-9cf92022ae75", id)
	assert.Nil(t, data)

	s.AssertExpectations(t)
}

func TestECSEnvironment_Update_RequiresReplacement(t *testing.T) {
	s := new(mockEnvironmentStore)
	p := newECSEnvironmentProvisioner(&ECSEnvironmentResource{
		environmentStore: s,
	})

	s.On("store", []*ecs.KeyValuePair{
		{
			Name:  aws.String("FOO"),
			Value: aws.String("bar"),
		},
		{
			Name:  aws.String("BAR"),
			Value: aws.String("foo"),
		},
	}).Return("56152438-5fef-4c96-bbe1-9cf92022ae75", nil)

	id, data, err := p.Provision(ctx, customresources.Request{
		RequestType:        customresources.Update,
		PhysicalResourceId: "56152438-5fef-4c96-bbe1-9cf92022ae75",
		ResourceProperties: &ECSEnvironmentProperties{
			Environment: []*ecs.KeyValuePair{
				{
					Name:  aws.String("FOO"),
					Value: aws.String("bar"),
				},
				{
					Name:  aws.String("BAR"),
					Value: aws.String("foo"),
				},
			},
		},
		OldResourceProperties: &ECSEnvironmentProperties{
			Environment: []*ecs.KeyValuePair{
				{
					Name:  aws.String("BAR"),
					Value: aws.String("foo"),
				},
				{
					Name:  aws.String("FOO"),
					Value: aws.String("bar"),
				},
			},
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, "56152438-5fef-4c96-bbe1-9cf92022ae75", id)
	assert.Nil(t, data)

	s.AssertExpectations(t)
}

func TestServiceRequiresReplacement(t *testing.T) {
	tests := []struct {
		new, old properties
		out      bool
	}{
		{
			&ECSServiceProperties{Cluster: aws.String("cluster"), TaskDefinition: aws.String("td:2"), DesiredCount: customresources.Int(1)},
			&ECSServiceProperties{Cluster: aws.String("cluster"), TaskDefinition: aws.String("td:1"), DesiredCount: customresources.Int(0)},
			false,
		},

		{
			&ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: customresources.Int(8080), LoadBalancerName: aws.String("elb")}}},
			&ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: customresources.Int(8080), LoadBalancerName: aws.String("elb")}}},
			false,
		},

		// Can't change clusters.
		{
			&ECSServiceProperties{Cluster: aws.String("clusterB")},
			&ECSServiceProperties{Cluster: aws.String("clusterA")},
			true,
		},

		// Can't change name.
		{
			&ECSServiceProperties{ServiceName: aws.String("acme-inc-B")},
			&ECSServiceProperties{ServiceName: aws.String("acme-inc-A")},
			true,
		},

		// Can't change role.
		{
			&ECSServiceProperties{Role: aws.String("roleB")},
			&ECSServiceProperties{Role: aws.String("roleA")},
			true,
		},

		// Can't change load balancers
		{
			&ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: customresources.Int(8080), LoadBalancerName: aws.String("elbB")}}},
			&ECSServiceProperties{LoadBalancers: []LoadBalancer{{ContainerName: aws.String("web"), ContainerPort: customresources.Int(8080), LoadBalancerName: aws.String("elbA")}}},
			true,
		},
	}

	for _, tt := range tests {
		out, err := requiresReplacement(tt.new, tt.old)
		assert.NoError(t, err)
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

func (m *mockECS) RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.RegisterTaskDefinitionOutput), args.Error(1)
}

type mockEnvironmentStore struct {
	mock.Mock
}

func (m *mockEnvironmentStore) store(env []*ecs.KeyValuePair) (string, error) {
	args := m.Called(env)
	return args.String(0), args.Error(1)
}

func (m *mockEnvironmentStore) fetch(id string) ([]*ecs.KeyValuePair, error) {
	args := m.Called(id)
	return args.Get(0).([]*ecs.KeyValuePair), args.Error(1)
}
