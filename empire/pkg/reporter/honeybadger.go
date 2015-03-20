package reporter

import (
	"errors"

	"github.com/jcoene/honeybadger"
	"golang.org/x/net/context"
)

// HoneybadgerReporter is a Reporter implementation backed for Honeybadger.
type HoneybadgerReporter struct{}

func (h *HoneybadgerReporter) Report(ctx context.Context, err error) error {
	if honeybadger.ApiKey == "" {
		return errors.New("missing honeybadger.ApiKey")
	}

	report, err2 := honeybadger.NewReport(err)
	if err2 != nil {
		return err2
	}

	if err2 := report.Send(); err2 != nil {
		return err2
	}

	return nil
}
