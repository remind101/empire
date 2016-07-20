package cloudformation

import (
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

var (
	DefaultVisibilityHeartbeat = 1 * time.Minute
	DefaultNumWorkers          = 10
)

// sqsClient duck types the sqs.SQS interface.
type sqsClient interface {
	ReceiveMessage(*sqs.ReceiveMessageInput) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(*sqs.DeleteMessageInput) (*sqs.DeleteMessageOutput, error)
	ChangeMessageVisibility(*sqs.ChangeMessageVisibilityInput) (*sqs.ChangeMessageVisibilityOutput, error)
}

// SQSDispatcher pulls messages from SQS, and dispatches them to a handler.
type SQSDispatcher struct {
	// Root context.Context to use. If a reporter.Reporter is embedded,
	// errors generated will be reporter there. If a logger.Logger is
	// embedded, logging will be logged there.
	Context context.Context

	// The SQS queue url to listen for CloudFormation Custom Resource
	// requests.
	QueueURL string

	// When a message is pulled off of sqs, the visibility timeout will be
	// extended by this much, and periodically extended while the handler
	// performs it's work. If this process crashes, then the sqs message
	// will be redelivered later.
	VisibilityHeartbeat time.Duration

	// Number of worker goroutines to start for receiving messages.
	NumWorkers int

	sqs sqsClient
}

func newSQSDispatcher(config client.ConfigProvider) *SQSDispatcher {
	return &SQSDispatcher{
		VisibilityHeartbeat: DefaultVisibilityHeartbeat,
		NumWorkers:          DefaultNumWorkers,
		sqs:                 sqs.New(config),
	}
}

// Start starts multiple goroutines pulling messages off of the queue.
func (q *SQSDispatcher) Start(handle func(context.Context, *sqs.Message) error) {
	for i := 0; i < q.NumWorkers; i++ {
		go q.start(handle)
	}
}

// start starts a pulling messages off of the queue and passing them to the
// handler.
func (q *SQSDispatcher) start(handle func(context.Context, *sqs.Message) error) {
	for {
		ctx := q.Context

		resp, err := q.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
			QueueUrl: aws.String(q.QueueURL),
		})
		if err != nil {
			reporter.Report(ctx, err)
			continue
		}

		for _, m := range resp.Messages {
			go func(m *sqs.Message) {
				if err := q.handle(ctx, handle, m); err != nil {
					reporter.Report(ctx, err)
				}
			}(m)
		}
	}
}

func (q *SQSDispatcher) handle(ctx context.Context, handle func(context.Context, *sqs.Message) error, message *sqs.Message) (err error) {
	defer func() {
		if err == nil {
			_, err = q.sqs.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(q.QueueURL),
				ReceiptHandle: message.ReceiptHandle,
			})
		}
	}()

	visibilityTimeout := int64(float64(q.VisibilityHeartbeat) / float64(time.Second))

	_, err = q.sqs.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(q.QueueURL),
		ReceiptHandle:     message.ReceiptHandle,
		VisibilityTimeout: aws.Int64(visibilityTimeout),
	})

	errCh := make(chan error)
	go func() {
		errCh <- handle(ctx, message)
	}()

	tick := time.Tick(q.VisibilityHeartbeat / 2)

	for {
		select {
		case err = <-errCh:
			return
		case <-tick:
			_, err = q.sqs.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
				QueueUrl:          aws.String(q.QueueURL),
				ReceiptHandle:     message.ReceiptHandle,
				VisibilityTimeout: aws.Int64(visibilityTimeout),
			})
			if err != nil {
				return
			}
		}
	}
}
