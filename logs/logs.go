package logs

import (
	"fmt"
	"io"
	"time"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/uuid"
	"github.com/remind101/kinesumer"
)

type KinesisLogsStreamer struct{}

func NewKinesisLogsStreamer() *KinesisLogsStreamer {
	return &KinesisLogsStreamer{}
}

func (s *KinesisLogsStreamer) StreamLogs(app *empire.App, w io.Writer, duration time.Duration) error {
	k, err := kinesumer.NewDefault(app.ID, duration)
	if err != nil {
		return fmt.Errorf("error initializing kinesumer: %v", err)
	}

	_, err = k.Begin()
	if err != nil {
		return fmt.Errorf("error starting kinesumer: %v", err)
	}
	defer k.End()

	for {
		rec := <-k.Records()
		msg := append(rec.Data(), '\n')
		if _, err := w.Write(msg); err != nil {
			return fmt.Errorf("error writing kinesis record to log stream: %v", err)
		}
	}
}

// RecordToCloudWatch returns a RunRecorder that writes the log record to
// CloudWatch Logs.
func RecordToCloudWatch(group string, config client.ConfigProvider) empire.RunRecorder {
	c := cloudwatchlogs.New(config)
	g := cloudwatch.NewGroup(group, c)
	return func() (io.Writer, error) {
		stream := uuid.New()
		w, err := g.Create(stream)
		if err != nil {
			return nil, err
		}

		url := fmt.Sprintf("https://console.aws.amazon.com/cloudwatch/home?region=%s#logEvent:group=%s;stream=%s", *c.Config.Region, group, stream)
		return &writerWithURL{w, url}, nil
	}
}

// writerWithURL is an io.Writer that has a URL() method.
type writerWithURL struct {
	io.Writer
	url string
}

// URL returns the underyling url.
func (w *writerWithURL) URL() string {
	return w.url
}

// RecordTo returns a RunRecorder that writes the log record to the io.Writer
func RecordTo(w io.Writer) empire.RunRecorder {
	return func() (io.Writer, error) {
		return w, nil
	}
}
