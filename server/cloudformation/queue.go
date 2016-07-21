package cloudformation

import (
	"fmt"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/sqs"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

var (
	defaultVisibilityHeartbeat = 1 * time.Minute
	defaultNumWorkers          = 10
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

	stopped chan struct{}
	after   func(time.Duration) <-chan time.Time
	sqs     sqsClient
}

func newSQSDispatcher(config client.ConfigProvider) *SQSDispatcher {
	return &SQSDispatcher{
		VisibilityHeartbeat: defaultVisibilityHeartbeat,
		NumWorkers:          defaultNumWorkers,
		after:               time.After,
		sqs:                 sqs.New(config),
	}
}

// Start starts multiple goroutines pulling messages off of the queue.
func (q *SQSDispatcher) Start(handle func(context.Context, *sqs.Message) error) {
	var wg sync.WaitGroup
	for i := 0; i < q.NumWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			q.start(handle)
		}()
	}

	wg.Wait()
}

func (q *SQSDispatcher) Stop() {
	close(q.stopped)
}

// start starts a pulling messages off of the queue and passing them to the
// handler.
func (q *SQSDispatcher) start(handle func(context.Context, *sqs.Message) error) {
	var wg sync.WaitGroup
	for {
		select {
		case <-q.stopped:
			wg.Wait()
			return
		default:
			ctx := q.Context

			resp, err := q.sqs.ReceiveMessage(&sqs.ReceiveMessageInput{
				QueueUrl: aws.String(q.QueueURL),
			})
			if err != nil {
				reporter.Report(ctx, err)
				continue
			}

			for _, m := range resp.Messages {
				wg.Add(1)
				go func(m *sqs.Message) {
					defer wg.Done()
					if err := q.handle(ctx, handle, m); err != nil {
						reporter.Report(ctx, err)
					}
				}(m)
			}
		}
	}
}

func (q *SQSDispatcher) handle(ctx context.Context, handle func(context.Context, *sqs.Message) error, message *sqs.Message) (err error) {
	ctx, cancel := context.WithCancel(ctx)

	defer func() {
		if err == nil {
			_, err = q.sqs.DeleteMessage(&sqs.DeleteMessageInput{
				QueueUrl:      aws.String(q.QueueURL),
				ReceiptHandle: message.ReceiptHandle,
			})
		}
	}()

	var t <-chan time.Time
	t, err = q.extendMessageVisibilityTimeout(message.ReceiptHandle)

	errCh := make(chan error)
	go func() { errCh <- handle(ctx, message) }()

	for {
		select {
		case err = <-errCh:
			return
		case <-q.stopped:
			cancel()
		case <-t:
			t, err = q.extendMessageVisibilityTimeout(message.ReceiptHandle)
			if err != nil {
				return
			}
		}
	}
}

// extendMessageVisibilityTimeout extends the messages visibility timeout by
// VisibilityHeartbeat, and returns a channel that will receive after half of
// VisibilityTimeout has elapsed.
func (q *SQSDispatcher) extendMessageVisibilityTimeout(receiptHandle *string) (<-chan time.Time, error) {
	visibilityTimeout := int64(float64(q.VisibilityHeartbeat) / float64(time.Second))

	_, err := q.sqs.ChangeMessageVisibility(&sqs.ChangeMessageVisibilityInput{
		QueueUrl:          aws.String(q.QueueURL),
		ReceiptHandle:     receiptHandle,
		VisibilityTimeout: aws.Int64(visibilityTimeout),
	})
	if err != nil {
		return nil, fmt.Errorf("error extending message visibility timeout: %v", err)
	}

	return q.after(q.VisibilityHeartbeat / 2), nil
}
