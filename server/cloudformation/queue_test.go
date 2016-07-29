package cloudformation

import (
	"errors"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

func TestSQSDispatcher_Handle(t *testing.T) {
	s := new(mockSQSClient)
	q := &SQSDispatcher{
		VisibilityHeartbeat: defaultVisibilityHeartbeat,
		QueueURL:            "https://sqs.amazonaws.com",
		sqs:                 s,
		after: func(d time.Duration) <-chan time.Time {
			return nil
		},
	}

	s.On("ChangeMessageVisibility", &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String("https://sqs.amazonaws.com"),
		VisibilityTimeout: aws.Int64(60),
		ReceiptHandle:     aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	}).Return(&sqs.ChangeMessageVisibilityOutput{}, nil).Once()

	s.On("DeleteMessage", &sqs.DeleteMessageInput{
		QueueUrl:      aws.String("https://sqs.amazonaws.com"),
		ReceiptHandle: aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	}).Return(&sqs.DeleteMessageOutput{}, nil)

	handle := func(ctx context.Context, message *sqs.Message) error {
		return nil
	}
	err := q.handle(ctx, handle, &sqs.Message{
		ReceiptHandle: aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

func TestSQSDispatcher_Handle_ChangeMessageVisibility(t *testing.T) {
	s := new(mockSQSClient)
	q := &SQSDispatcher{
		VisibilityHeartbeat: defaultVisibilityHeartbeat,
		QueueURL:            "https://sqs.amazonaws.com",
		sqs:                 s,
		after: func(d time.Duration) <-chan time.Time {
			return nil
		},
	}

	awsErr := awserr.New("AccessDenied", "Stack with id acme-inc does not exist", errors.New(""))
	s.On("ChangeMessageVisibility", &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String("https://sqs.amazonaws.com"),
		VisibilityTimeout: aws.Int64(60),
		ReceiptHandle:     aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	}).Return(&sqs.ChangeMessageVisibilityOutput{}, awsErr).Once()

	handle := func(ctx context.Context, message *sqs.Message) error {
		return nil
	}
	err := q.handle(ctx, handle, &sqs.Message{
		ReceiptHandle: aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	})
	assert.Equal(t, awsErr, err)

	s.AssertExpectations(t)
}

func TestSQSDispatcher_Handle_Long(t *testing.T) {
	timeout := make(chan time.Time)
	s := new(mockSQSClient)
	q := &SQSDispatcher{
		VisibilityHeartbeat: defaultVisibilityHeartbeat,
		QueueURL:            "https://sqs.amazonaws.com",
		sqs:                 s,
		after: func(d time.Duration) <-chan time.Time {
			return timeout
		},
	}

	s.On("ChangeMessageVisibility", &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String("https://sqs.amazonaws.com"),
		VisibilityTimeout: aws.Int64(60),
		ReceiptHandle:     aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	}).Return(&sqs.ChangeMessageVisibilityOutput{}, nil).Twice()

	s.On("DeleteMessage", &sqs.DeleteMessageInput{
		QueueUrl:      aws.String("https://sqs.amazonaws.com"),
		ReceiptHandle: aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	}).Return(&sqs.DeleteMessageOutput{}, nil)

	handle := func(ctx context.Context, message *sqs.Message) error {
		timeout <- time.Now()
		return nil
	}
	err := q.handle(ctx, handle, &sqs.Message{
		ReceiptHandle: aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	})
	assert.NoError(t, err)

	s.AssertExpectations(t)
}

func TestSQSDispatcher_Handle_Error(t *testing.T) {
	s := new(mockSQSClient)
	q := &SQSDispatcher{
		VisibilityHeartbeat: defaultVisibilityHeartbeat,
		QueueURL:            "https://sqs.amazonaws.com",
		sqs:                 s,
		after: func(d time.Duration) <-chan time.Time {
			return nil
		},
	}

	s.On("ChangeMessageVisibility", &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String("https://sqs.amazonaws.com"),
		VisibilityTimeout: aws.Int64(60),
		ReceiptHandle:     aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	}).Return(&sqs.ChangeMessageVisibilityOutput{}, nil).Once()

	handle := func(ctx context.Context, message *sqs.Message) error {
		return errors.New("error uploading response")
	}
	err := q.handle(ctx, handle, &sqs.Message{
		ReceiptHandle: aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	})
	assert.Error(t, err)

	s.AssertExpectations(t)
}

func TestSQSDispatcher_Handle_Stopped(t *testing.T) {
	s := new(mockSQSClient)
	q := &SQSDispatcher{
		VisibilityHeartbeat: defaultVisibilityHeartbeat,
		QueueURL:            "https://sqs.amazonaws.com",
		sqs:                 s,
		stopped:             make(chan struct{}),
		after: func(d time.Duration) <-chan time.Time {
			return nil
		},
	}
	q.Stop()

	s.On("ChangeMessageVisibility", &sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String("https://sqs.amazonaws.com"),
		VisibilityTimeout: aws.Int64(60),
		ReceiptHandle:     aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	}).Return(&sqs.ChangeMessageVisibilityOutput{}, nil).Once()

	handle := func(ctx context.Context, message *sqs.Message) error {
		<-ctx.Done()
		return ctx.Err()
	}
	err := q.handle(ctx, handle, &sqs.Message{
		ReceiptHandle: aws.String("MbZj6wDWli+JvwwJaBV+3dcjk2YW2vA3+STFFljTM8tJJg6HRG6PYSasuWXPJB+CwLj1FjgXUv1uSj1gUPAWV66FU/WeR4mq2OKpEGYWbnLmpRCJVAyeMjeU5ZBdtcQ+QEauMZc8ZRv37sIW2iJKq3M9MFx1YvV11A2x/KSbkJ0="),
	})
	assert.Equal(t, context.Canceled, err)

	s.AssertExpectations(t)
}
