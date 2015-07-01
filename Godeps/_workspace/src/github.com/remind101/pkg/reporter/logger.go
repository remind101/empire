package reporter

import (
	"fmt"

	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

// LogReporter is a Handler that logs the error to a log.Logger.
type LogReporter struct{}

func NewLogReporter() *LogReporter {
	return &LogReporter{}
}

// Report logs the error to the Logger.
func (h *LogReporter) Report(ctx context.Context, err error) error {
	switch err := err.(type) {
	case *Error:
		var line *BacktraceLine

		if len(err.Backtrace) > 0 {
			line = err.Backtrace[0]
		} else {
			line = &BacktraceLine{
				File: "unknown",
				Line: 0,
			}
		}

		logger.Error(ctx, "", "error", fmt.Sprintf(`"%v"`, err), "line", line.Line, "file", line.File)
	default:
		logger.Error(ctx, "", "error", fmt.Sprintf(`"%v"`, err))
	}

	return nil
}
