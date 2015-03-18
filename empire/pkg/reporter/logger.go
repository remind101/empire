package reporter

import (
	"io/ioutil"
	"log"

	"golang.org/x/net/context"
)

// LogReporter is a Handler that logs the error to a log.Logger.
type LogReporter struct {
	*log.Logger
}

func NewLogReporter(l *log.Logger) *LogReporter {
	return &LogReporter{
		Logger: l,
	}
}

// Report logs the error to the Logger.
func (h *LogReporter) Report(ctx context.Context, err error) error {
	h.logger().Println(err)
	return nil
}

// defaultLogger is the default logger to use when one is not defined. It simply
// writes to /dev/null.
var defaultLogger = log.New(ioutil.Discard, "", log.LstdFlags)

func (h *LogReporter) logger() *log.Logger {
	if h.Logger == nil {
		return defaultLogger
	}

	return h.Logger
}
