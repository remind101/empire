package logs

import (
	"fmt"
	"io"
	"time"

	"github.com/remind101/empire"
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
