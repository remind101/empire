package ecs

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/aws/aws-sdk-go/service/s3"
	docker "github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestTaskEngine_Run_Detached(t *testing.T) {
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &TaskEngine{
		Cluster:        "cluster",
		ecs:            e,
		cloudformation: c,
	}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("TaskDefinitions"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-webTaskDefinition-PVBIR7PA0DV7:1"),
					},
				},
			},
		},
	}, nil)

	e.On("RunTask", &ecs.RunTaskInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-webTaskDefinition-PVBIR7PA0DV7:1"),
		Cluster:        aws.String("cluster"),
		Count:          aws.Int64(1),
		StartedBy:      aws.String("acme-inc"),
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				{
					Name:    aws.String("web"),
					Command: []*string{aws.String("./bin/web")},
				},
			},
		},
	}).Return(&ecs.RunTaskOutput{
		Tasks: []*ecs.Task{
			&ecs.Task{
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
				DesiredStatus:        aws.String("RUNNING"),
				LastStatus:           aws.String("PENDING"),
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-webTaskDefinition-PVBIR7PA0DV7:1"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64"),
			},
		},
	}, nil)

	err := s.Run(context.Background(), &empire.App{
		Name: "acme-inc",
		Formation: empire.Formation{
			"web": empire.Process{
				Command: empire.Command{"./bin/web"},
			},
		},
	}, nil)
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

func TestTaskEngine_Run_Attached(t *testing.T) {
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	x := new(mockEC2Client)
	d := new(mockDockerClient)
	s := &TaskEngine{
		Cluster: "cluster",
		NewDockerClient: func(ec2Instance *ec2.Instance) (DockerClient, error) {
			return d, nil
		},
		ecs:            e,
		cloudformation: c,
		ec2:            x,
	}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("TaskDefinitions"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-webTaskDefinition-PVBIR7PA0DV7:1"),
					},
				},
			},
		},
	}, nil)

	e.On("RunTask", &ecs.RunTaskInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-webTaskDefinition-PVBIR7PA0DV7:1"),
		Cluster:        aws.String("cluster"),
		Count:          aws.Int64(1),
		StartedBy:      aws.String("acme-inc"),
		Overrides: &ecs.TaskOverride{
			ContainerOverrides: []*ecs.ContainerOverride{
				{
					Name:    aws.String("web"),
					Command: []*string{aws.String("./bin/web")},
					Environment: []*ecs.KeyValuePair{
						{Name: aws.String("TERM"), Value: aws.String("xterm")},
						{Name: aws.String("ECS_DOCKER_CONFIG_TTY"), Value: aws.String("true")},
						{Name: aws.String("ECS_DOCKER_CONFIG_OPEN_STDIN"), Value: aws.String("true")},
					},
				},
			},
		},
	}).Return(&ecs.RunTaskOutput{
		Tasks: []*ecs.Task{
			&ecs.Task{
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
				DesiredStatus:        aws.String("RUNNING"),
				LastStatus:           aws.String("PENDING"),
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-webTaskDefinition-PVBIR7PA0DV7:1"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64"),
			},
		},
	}, nil)

	e.On("StopTask", &ecs.StopTaskInput{
		Cluster: aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
		Task:    aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"),
	}).Return(&ecs.StopTaskOutput{}, nil)

	e.On("DescribeContainerInstances", &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
		ContainerInstances: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64")},
	}).Return(&ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			&ecs.ContainerInstance{
				AgentConnected:       aws.Bool(true),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/4c543eed-f83f-47da-b1d8-3d23f1da4c64"),
				Ec2InstanceId:        aws.String("i-042f39dc"),
			},
		},
	}, nil)

	x.On("DescribeInstances", &ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String("i-042f39dc")},
	}).Return(&ec2.DescribeInstancesOutput{
		Reservations: []*ec2.Reservation{
			&ec2.Reservation{
				Instances: []*ec2.Instance{
					&ec2.Instance{
						InstanceId:       aws.String("i-042f39dc"),
						PrivateIpAddress: aws.String("192.168.1.88"),
					},
				},
			},
		},
	}, nil)

	e.On("WaitUntilTasksNotPending", &ecs.DescribeTasksInput{
		Cluster: aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster"),
		Tasks:   []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb")},
	}).Return(nil)

	d.On("ListContainers", docker.ListContainersOptions{
		All: true,
		Filters: map[string][]string{
			"label": []string{"com.amazonaws.ecs.task-arn=arn:aws:ecs:us-east-1:012345678910:task/fdf2c302-468c-4e55-b884-5331d816e7fb"},
		},
	}).Return([]docker.APIContainers{
		docker.APIContainers{
			ID: "4c01db0b339c",
		},
	}, nil)

	stdin := strings.NewReader("ls\n")
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	d.On("AttachToContainer", docker.AttachToContainerOptions{
		Container:    "4c01db0b339c",
		InputStream:  stdin,
		OutputStream: stdout,
		ErrorStream:  stderr,
		Logs:         true,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	}).Return(nil)

	stdio := &empire.IO{
		Stdin:  stdin,
		Stdout: stdout,
		Stderr: stderr,
	}

	err := s.Run(context.Background(), &empire.App{
		Name: "acme-inc",
		Formation: empire.Formation{
			"web": empire.Process{
				Command: empire.Command{"./bin/web"},
				Environment: map[string]string{
					"TERM": "xterm",
				},
			},
		},
	}, stdio)
	assert.NoError(t, err)
	assert.Equal(t, "Attaching to task/fdf2c302-468c-4e55-b884-5331d816e7fb...\r\n", stderr.String())

	e.AssertExpectations(t)
	c.AssertExpectations(t)
	x.AssertExpectations(t)
	d.AssertExpectations(t)
}

func TestTaskEngine_Tasks(t *testing.T) {
	c := new(mockCloudFormationClient)
	e := new(mockECSClient)
	s := &TaskEngine{
		Cluster:        "cluster",
		ecs:            e,
		cloudformation: c,
	}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{
				StackStatus: aws.String("CREATE_COMPLETE"),
				Outputs: []*cloudformation.Output{
					{
						OutputKey:   aws.String("Services"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:service/acme-inc-web"),
					},
					{
						OutputKey:   aws.String("TaskDefinitions"),
						OutputValue: aws.String("web=arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-webTaskDefinition-PVBIR7PA0DV7:1"),
					},
				},
			},
		},
	}, nil)

	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:     aws.String("cluster"),
		ServiceName: aws.String("acme-inc-web"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
		},
	}, nil)

	e.On("ListTasksPages", &ecs.ListTasksInput{
		Cluster:   aws.String("cluster"),
		StartedBy: aws.String("acme-inc"),
	}).Return(&ecs.ListTasksOutput{
		TaskArns: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
		},
	}, nil)

	dt := time.Date(2009, time.November, 10, 23, 0, 0, 0, time.UTC)
	e.On("DescribeTasks", &ecs.DescribeTasksInput{
		Cluster: aws.String("cluster"),
		Tasks: []*string{
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
			aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
		},
	}).Return(&ecs.DescribeTasksOutput{
		Tasks: []*ecs.Task{
			{
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/0b69d5c0-d655-4695-98cd-5d2d526d9d5a"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-web:0"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-1"),
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-1"),
				LastStatus:           aws.String("RUNNING"),
				StartedAt:            &dt,
			},
			{
				TaskArn:              aws.String("arn:aws:ecs:us-east-1:012345678910:task/c09f0188-7f87-4b0f-bfc3-16296622b6fe"),
				TaskDefinitionArn:    aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc--run:0"),
				ClusterArn:           aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-2"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-2"),
				LastStatus:           aws.String("PENDING"),
				CreatedAt:            &dt,
			},
		},
	}, nil)

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc-web:0"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name:   aws.String("web"),
					Cpu:    aws.Int64(256),
					Memory: aws.Int64(int64(256)),
				},
			},
		},
	}, nil)

	e.On("DescribeTaskDefinition", &ecs.DescribeTaskDefinitionInput{
		TaskDefinition: aws.String("arn:aws:ecs:us-east-1:012345678910:task-definition/acme-inc--run:0"),
	}).Return(&ecs.DescribeTaskDefinitionOutput{
		TaskDefinition: &ecs.TaskDefinition{
			ContainerDefinitions: []*ecs.ContainerDefinition{
				{
					Name:   aws.String("run"),
					Cpu:    aws.Int64(256),
					Memory: aws.Int64(int64(256)),
				},
			},
		},
	}, nil)

	e.On("DescribeContainerInstances", &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-1"),
		ContainerInstances: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-1")},
	}).Return(&ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			{
				Ec2InstanceId:        aws.String("ec2-instance-id-1"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-1"),
			},
		},
	}, nil)

	e.On("DescribeContainerInstances", &ecs.DescribeContainerInstancesInput{
		Cluster:            aws.String("arn:aws:ecs:us-east-1:012345678910:cluster/cluster-name-2"),
		ContainerInstances: []*string{aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-2")},
	}).Return(&ecs.DescribeContainerInstancesOutput{
		ContainerInstances: []*ecs.ContainerInstance{
			{
				Ec2InstanceId:        aws.String("ec2-instance-id-2"),
				ContainerInstanceArn: aws.String("arn:aws:ecs:us-east-1:012345678910:container-instance/container-instance-id-2"),
			},
		},
	}, nil)

	_, err := s.Tasks(context.Background(), &empire.App{
		Name: "acme-inc",
		Formation: empire.Formation{
			"web": empire.Process{
				Command: empire.Command{"./bin/web"},
			},
		},
	})
	assert.NoError(t, err)

	e.AssertExpectations(t)
	c.AssertExpectations(t)
}

type mockCloudFormationClient struct {
	cloudformationClient
	mock.Mock
}

func (m *mockCloudFormationClient) CreateStack(input *cloudformation.CreateStackInput) (*cloudformation.CreateStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) UpdateStack(input *cloudformation.UpdateStackInput) (*cloudformation.UpdateStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.UpdateStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) DeleteStack(input *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DeleteStackOutput), args.Error(1)
}

func (m *mockCloudFormationClient) DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStacksOutput), args.Error(1)
}

func (m *mockCloudFormationClient) ListStackResourcesPages(input *cloudformation.ListStackResourcesInput, fn func(*cloudformation.ListStackResourcesOutput, bool) bool) error {
	args := m.Called(input)
	fn(args.Get(0).(*cloudformation.ListStackResourcesOutput), true)
	return args.Error(1)
}

func (m *mockCloudFormationClient) DescribeStackResource(input *cloudformation.DescribeStackResourceInput) (*cloudformation.DescribeStackResourceOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStackResourceOutput), args.Error(1)
}

func (m *mockCloudFormationClient) WaitUntilStackCreateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

func (m *mockCloudFormationClient) WaitUntilStackUpdateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

func (m *mockCloudFormationClient) ValidateTemplate(input *cloudformation.ValidateTemplateInput) (*cloudformation.ValidateTemplateOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.ValidateTemplateOutput), args.Error(1)
}

type mockS3Client struct {
	mock.Mock
}

func (m *mockS3Client) PutObject(input *s3.PutObjectInput) (*s3.PutObjectOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*s3.PutObjectOutput), args.Error(1)
}

type mockECSClient struct {
	ecsClient
	mock.Mock
}

func (m *mockECSClient) ListTasksPages(input *ecs.ListTasksInput, fn func(p *ecs.ListTasksOutput, lastPage bool) (shouldContinue bool)) error {
	args := m.Called(input)
	fn(args.Get(0).(*ecs.ListTasksOutput), true)
	return args.Error(1)
}

func (m *mockECSClient) DescribeTasks(input *ecs.DescribeTasksInput) (*ecs.DescribeTasksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeTasksOutput), args.Error(1)
}

func (m *mockECSClient) DescribeTaskDefinition(input *ecs.DescribeTaskDefinitionInput) (*ecs.DescribeTaskDefinitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeTaskDefinitionOutput), args.Error(1)
}

func (m *mockECSClient) DescribeServices(input *ecs.DescribeServicesInput) (*ecs.DescribeServicesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeServicesOutput), args.Error(1)
}

func (m *mockECSClient) DescribeContainerInstances(input *ecs.DescribeContainerInstancesInput) (*ecs.DescribeContainerInstancesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.DescribeContainerInstancesOutput), args.Error(1)
}

func (m *mockECSClient) RegisterTaskDefinition(input *ecs.RegisterTaskDefinitionInput) (*ecs.RegisterTaskDefinitionOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.RegisterTaskDefinitionOutput), args.Error(1)
}

func (m *mockECSClient) RunTask(input *ecs.RunTaskInput) (*ecs.RunTaskOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.RunTaskOutput), args.Error(1)
}

func (m *mockECSClient) StopTask(input *ecs.StopTaskInput) (*ecs.StopTaskOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ecs.StopTaskOutput), args.Error(1)
}

func (m *mockECSClient) WaitUntilTasksNotPending(input *ecs.DescribeTasksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

type mockEC2Client struct {
	mock.Mock
}

func (m *mockEC2Client) DescribeInstances(input *ec2.DescribeInstancesInput) (*ec2.DescribeInstancesOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*ec2.DescribeInstancesOutput), args.Error(1)
}

type mockDockerClient struct {
	mock.Mock
}

func (m *mockDockerClient) ListContainers(options docker.ListContainersOptions) ([]docker.APIContainers, error) {
	args := m.Called(options)
	return args.Get(0).([]docker.APIContainers), args.Error(1)
}

func (m *mockDockerClient) AttachToContainer(options docker.AttachToContainerOptions) error {
	args := m.Called(options)
	return args.Error(0)
}
