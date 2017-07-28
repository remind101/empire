package empire

import (
	"io"
	"time"
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
