package cloudwatch

import (
	"io"
	"time"

	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

// Throttling and limits from http://docs.aws.amazon.com/AmazonCloudWatch/latest/DeveloperGuide/cloudwatch_limits.html
const (
	// The maximum rate of a GetLogEvents request is 10 requests per second per AWS account.
	readThrottle = time.Second / 10

	// The maximum rate of a PutLogEvents request is 5 requests per second per log stream.
	writeThrottle = time.Second / 5
)

// now is a function that returns the current time.Time. It's a variable so that
// it can be stubbed out in unit tests.
var now = time.Now

// client duck types the aws sdk client for testing.
type client interface {
	PutLogEvents(*cloudwatchlogs.PutLogEventsInput) (*cloudwatchlogs.PutLogEventsOutput, error)
	CreateLogStream(*cloudwatchlogs.CreateLogStreamInput) (*cloudwatchlogs.CreateLogStreamOutput, error)
	GetLogEvents(*cloudwatchlogs.GetLogEventsInput) (*cloudwatchlogs.GetLogEventsOutput, error)
}

// Group wraps a log stream group and provides factory methods for creating
// readers and writers for streams.
type Group struct {
	group  string
	client *cloudwatchlogs.CloudWatchLogs
}

// NewGroup returns a new Group instance.
func NewGroup(group string, client *cloudwatchlogs.CloudWatchLogs) *Group {
	return &Group{
		group:  group,
		client: client,
	}
}

// Create creates a log stream in the group and returns an io.Writer for it.
func (g *Group) Create(stream string) (io.Writer, error) {
	if _, err := g.client.CreateLogStream(&cloudwatchlogs.CreateLogStreamInput{
		LogGroupName:  &g.group,
		LogStreamName: &stream,
	}); err != nil {
		return nil, err
	}

	return NewWriter(g.group, stream, g.client), nil
}

// Open returns an io.Reader to read from the log stream.
func (g *Group) Open(stream string) (io.Reader, error) {
	return NewReader(g.group, stream, g.client), nil
}
