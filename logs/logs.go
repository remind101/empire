package logs

import (
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go/aws/client"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/ejholmes/cloudwatch"
	"github.com/remind101/empire"
	"github.com/remind101/empire/internal/uuid"
)

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
