package cloudformation

import (
	"errors"
	"html/template"
	"testing"

	"golang.org/x/net/context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestScheduler_Submit_NewStack(t *testing.T) {
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		cloudformation: c,
		stackName:      stackName,
	}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&cloudformation.DescribeStacksOutput{}, awserr.New("400", "Stack with id app-c9366591-ab68-4d49-a333-95ce5a23df68 does not exist", errors.New("")))

	c.On("CreateStack", &cloudformation.CreateStackInput{
		StackName:    aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
		TemplateBody: aws.String("{}"),
	}).Return(&cloudformation.CreateStackOutput{}, nil)

	c.On("WaitUntilStackCreateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(nil)

	err := s.Submit(context.Background(), &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestScheduler_Submit_ExistingStack(t *testing.T) {
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		cloudformation: c,
		stackName:      stackName,
	}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("CREATE_COMPLETE")},
		},
	}, nil)

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:    aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
		TemplateBody: aws.String("{}"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(nil)

	err := s.Submit(context.Background(), &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
	})
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func TestScheduler_Submit_StackUpdateInProgress(t *testing.T) {
	c := new(mockCloudFormationClient)
	s := &Scheduler{
		Template:       template.Must(template.New("t").Parse("{}")),
		Wait:           true,
		cloudformation: c,
		stackName:      stackName,
	}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackStatus: aws.String("UPDATE_IN_PROGRESS")},
		},
	}, nil)

	c.On("UpdateStack", &cloudformation.UpdateStackInput{
		StackName:    aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
		TemplateBody: aws.String("{}"),
	}).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("app-c9366591-ab68-4d49-a333-95ce5a23df68"),
	}).Return(nil).Twice()

	err := s.Submit(context.Background(), &scheduler.App{
		ID: "c9366591-ab68-4d49-a333-95ce5a23df68",
	})
	assert.NoError(t, err)

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

func (m *mockCloudFormationClient) DescribeStacks(input *cloudformation.DescribeStacksInput) (*cloudformation.DescribeStacksOutput, error) {
	args := m.Called(input)
	return args.Get(0).(*cloudformation.DescribeStacksOutput), args.Error(1)
}

func (m *mockCloudFormationClient) WaitUntilStackCreateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}

func (m *mockCloudFormationClient) WaitUntilStackUpdateComplete(input *cloudformation.DescribeStacksInput) error {
	args := m.Called(input)
	return args.Error(0)
}
