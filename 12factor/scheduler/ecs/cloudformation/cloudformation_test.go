package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStackBuilder_Services(t *testing.T) {
	c := new(mockCloudformationClient)
	b := &StackBuilder{
		cloudformation: c,
		stackName:      stackName,
	}

	c.On("ListStackResourcesPages", &cloudformation.ListStackResourcesInput{
		StackName: aws.String("app"),
	}).Return(&cloudformation.ListStackResourcesOutput{
		StackResourceSummaries: []*cloudformation.StackResourceSummary{
			{
				ResourceType:      aws.String("AWS::ECS::Service"),
				LogicalResourceId: aws.String("ProcessWeb"),
			},
			{
				ResourceType:      aws.String("AWS::ECS::Service"),
				LogicalResourceId: aws.String("ProcessWorker"),
			},
			{
				ResourceType:      aws.String("AWS::EC2::Instance"),
				LogicalResourceId: aws.String("Instance"),
			},
		},
	}, nil)

	c.On("DescribeStackResource", &cloudformation.DescribeStackResourceInput{
		StackName:         aws.String("app"),
		LogicalResourceId: aws.String("ProcessWeb"),
	}).Return(&cloudformation.DescribeStackResourceOutput{
		StackResourceDetail: &cloudformation.StackResourceDetail{
			PhysicalResourceId: aws.String("service--web"),
			Metadata:           aws.String(`{"Name":"web"}`),
		},
	}, nil)

	c.On("DescribeStackResource", &cloudformation.DescribeStackResourceInput{
		StackName:         aws.String("app"),
		LogicalResourceId: aws.String("ProcessWorker"),
	}).Return(&cloudformation.DescribeStackResourceOutput{
		StackResourceDetail: &cloudformation.StackResourceDetail{
			PhysicalResourceId: aws.String("service--worker"),
			Metadata:           aws.String(`{"Name":"worker"}`),
		},
	}, nil)

	services, err := b.Services("app")
	assert.NoError(t, err)
	assert.Equal(t, map[string]string{"web": "service--web", "worker": "service--worker"}, services)
}

type mockCloudformationClient struct {
	mock.Mock
}

func (c *mockCloudformationClient) CreateStack(input *cloudformation.CreateStackInput) (*cloudformation.CreateStackOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*cloudformation.CreateStackOutput), args.Error(1)
}

func (c *mockCloudformationClient) DeleteStack(input *cloudformation.DeleteStackInput) (*cloudformation.DeleteStackOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*cloudformation.DeleteStackOutput), args.Error(1)
}

func (c *mockCloudformationClient) ListStackResourcesPages(input *cloudformation.ListStackResourcesInput, fn func(*cloudformation.ListStackResourcesOutput, bool) bool) error {
	args := c.Called(input)
	fn(args.Get(0).(*cloudformation.ListStackResourcesOutput), true)
	return args.Error(1)
}

func (c *mockCloudformationClient) DescribeStackResource(input *cloudformation.DescribeStackResourceInput) (*cloudformation.DescribeStackResourceOutput, error) {
	args := c.Called(input)
	return args.Get(0).(*cloudformation.DescribeStackResourceOutput), args.Error(1)
}
