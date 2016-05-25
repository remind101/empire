package cloudformation

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestStackUpdateQueue_UpdateStack_EmptyQueue(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	c := new(mockCloudFormationClient)
	q := &stackUpdateQueue{
		cloudformationClient: c,
		db:                   db,
	}

	input := &cloudformation.UpdateStackInput{
		StackName: aws.String("acme-inc"),
	}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackName: aws.String("acme-inc")},
		},
	}, nil)

	c.On("UpdateStack", input).Return(&cloudformation.UpdateStackOutput{}, nil)

	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil)

	_, err := q.UpdateStack(input)
	assert.NoError(t, err)

	c.AssertExpectations(t)
}

func testStackUpdateQueue_UpdateStack_ActiveUpdate(t *testing.T) {
	db := newDB(t)
	defer db.Close()

	c := new(mockCloudFormationClient)
	q := &stackUpdateQueue{
		cloudformationClient: c,
		db:                   db,
	}

	inputA := &cloudformation.UpdateStackInput{
		StackName: aws.String("acme-inc"),
		Parameters: []*cloudformation.Parameter{
			{ParameterKey: aws.String("A"), ParameterValue: aws.String("value")},
		},
	}

	//inputB := &cloudformation.UpdateStackInput{
	//StackName: aws.String("acme-inc"),
	//Parameters: []*cloudformation.Parameter{
	//{ParameterKey: aws.String("A"), ParameterValue: aws.String("value")},
	//},
	//}

	c.On("DescribeStacks", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(&cloudformation.DescribeStacksOutput{
		Stacks: []*cloudformation.Stack{
			{StackName: aws.String("acme-inc")},
		},
	}, nil).Twice()

	c.On("UpdateStack", inputA).Return(&cloudformation.UpdateStackOutput{}, nil)

	startedA := make(chan struct{})
	doneA := make(chan struct{})
	c.On("WaitUntilStackUpdateComplete", &cloudformation.DescribeStacksInput{
		StackName: aws.String("acme-inc"),
	}).Return(nil).Run(func(args mock.Arguments) {
		close(startedA)
		<-doneA
	})

	errCh := make(chan error)
	go func() {
		_, err := q.UpdateStack(inputA)
		errCh <- err
	}()

	// Let the first stack update complete. The second should start
	<-startedA
	close(doneA) // Let the first stack update complete.
	assert.Nil(t, <-errCh)

	c.AssertExpectations(t)
}
