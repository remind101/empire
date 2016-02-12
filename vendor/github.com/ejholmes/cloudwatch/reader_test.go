package cloudwatch

import (
	"bytes"
	"errors"
	"io"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/stretchr/testify/assert"
)

func TestReader(t *testing.T) {
	c := new(mockClient)
	r := &Reader{
		group:  aws.String("group"),
		stream: aws.String("1234"),
		client: c,
	}

	c.On("GetLogEvents", &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
	}).Once().Return(&cloudwatchlogs.GetLogEventsOutput{
		Events: []*cloudwatchlogs.OutputLogEvent{
			{Message: aws.String("Hello"), Timestamp: aws.Int64(1000)},
		},
	}, nil)

	err := r.read()
	assert.NoError(t, err)

	b := make([]byte, 1000)
	n, err := r.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	c.AssertExpectations(t)
}

func TestReader_Buffering(t *testing.T) {
	c := new(mockClient)
	r := &Reader{
		group:  aws.String("group"),
		stream: aws.String("1234"),
		client: c,
	}

	c.On("GetLogEvents", &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
	}).Once().Return(&cloudwatchlogs.GetLogEventsOutput{
		Events: []*cloudwatchlogs.OutputLogEvent{
			{Message: aws.String("Hello"), Timestamp: aws.Int64(1000)},
		},
	}, nil)

	err := r.read()
	assert.NoError(t, err)

	b := make([]byte, 3)
	n, err := r.Read(b) //Hel
	assert.NoError(t, err)
	assert.Equal(t, 3, n)

	n, err = r.Read(b) //lo
	assert.NoError(t, err)
	assert.Equal(t, 2, n)

	c.AssertExpectations(t)
}

func TestReader_EndOfFile(t *testing.T) {
	c := new(mockClient)
	r := &Reader{
		group:  aws.String("group"),
		stream: aws.String("1234"),
		client: c,
	}

	c.On("GetLogEvents", &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
	}).Once().Return(&cloudwatchlogs.GetLogEventsOutput{
		Events: []*cloudwatchlogs.OutputLogEvent{
			{Message: aws.String("Hello"), Timestamp: aws.Int64(1000)},
		},
		NextForwardToken: aws.String("next"),
	}, nil)

	c.On("GetLogEvents", &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
		NextToken:     aws.String("next"),
	}).Once().Return(&cloudwatchlogs.GetLogEventsOutput{
		Events: []*cloudwatchlogs.OutputLogEvent{
			{Message: aws.String("World"), Timestamp: aws.Int64(1000)},
		},
	}, nil)

	c.On("GetLogEvents", &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
		NextToken:     aws.String("next"),
	}).Once().Return(&cloudwatchlogs.GetLogEventsOutput{
		Events: []*cloudwatchlogs.OutputLogEvent{},
	}, nil)

	err := r.read()
	assert.NoError(t, err)

	b := make([]byte, 5)
	n, err := r.Read(b) //Hello
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	err = r.read()
	assert.NoError(t, err)

	n, err = r.Read(b) //World
	assert.NoError(t, err)
	assert.Equal(t, 5, n)

	err = r.read()
	assert.NoError(t, err)

	// Attempt to read more data, but there is none.
	n, err = r.Read(b)
	assert.NoError(t, err)
	assert.Equal(t, 0, n)

	c.AssertExpectations(t)
}

func TestReader_Err(t *testing.T) {
	c := new(mockClient)

	errBoom := errors.New("boom")
	c.On("GetLogEvents", &cloudwatchlogs.GetLogEventsInput{
		LogGroupName:  aws.String("group"),
		LogStreamName: aws.String("1234"),
	}).Once().Return(&cloudwatchlogs.GetLogEventsOutput{
		Events: []*cloudwatchlogs.OutputLogEvent{
			{Message: aws.String("Hello"), Timestamp: aws.Int64(1000)},
		},
	}, errBoom)

	r := newReader("group", "1234", c)

	b := new(bytes.Buffer)
	_, err := io.Copy(b, r)
	assert.Equal(t, errBoom, err)
}
