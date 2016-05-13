package empire

import (
	"io"
	"time"

	"github.com/remind101/kinesumer"
)

type LogsStreamer interface {
	StreamLogs(*App, io.Writer, time.Duration) error
}

var logsDisabled = &nullLogsStreamer{}

type nullLogsStreamer struct{}

func (s *nullLogsStreamer) StreamLogs(app *App, w io.Writer, duration time.Duration) error {
	io.WriteString(w, "Logs are disabled\n")
	return nil
}

type KinesisLogsStreamer struct{}

func NewKinesisLogsStreamer() *KinesisLogsStreamer {
	return &KinesisLogsStreamer{}
}

func (s *KinesisLogsStreamer) StreamLogs(app *App, w io.Writer, duration time.Duration) error {
	k, err := kinesumer.NewDefault(app.ID, duration)
	if err != nil {
		return err
	}

	_, err = k.Begin()
	if err != nil {
		return err
	}
	defer k.End()

	for {
		rec := <-k.Records()
		msg := append(rec.Data(), '\n')
		if _, err := w.Write(msg); err != nil {
			return err
		}
	}
}
