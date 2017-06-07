package reporter

import (
	"fmt"

	"github.com/pkg/errors"
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
	var file, line string
	var stack errors.StackTrace

	if err_with_stack, ok := err.(stackTracer); ok {
		stack = err_with_stack.StackTrace()
	}
	if stack != nil && len(stack) > 0 {
		file = fmt.Sprintf("%s", stack[0])
		line = fmt.Sprintf("%d", stack[0])
	} else {
		file = "unknown"
		line = "0"
	}

	logger.Error(ctx, "", "error", fmt.Sprintf(`"%v"`, err), "line", line, "file", file)
	return nil
}
