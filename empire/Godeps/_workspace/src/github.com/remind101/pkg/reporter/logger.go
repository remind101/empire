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
	logger.Log(ctx, "error", fmt.Sprintf(`"%v"`, err))
	return nil
}
