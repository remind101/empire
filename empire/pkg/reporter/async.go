package reporter

import "golang.org/x/net/context"

// AsyncReporter is a Reporter implementation that wraps another Reporter to called
// Report asynchronously.
type AsyncReporter struct {
	reporter Reporter
}

func Async(reporter Reporter) *AsyncReporter {
	return &AsyncReporter{
		reporter: reporter,
	}
}

// TODO Creating a new go routine for every Report call may not be the best
// thing, but should be ok for now.
func (r *AsyncReporter) Report(ctx context.Context, err error) error {
	go r.reporter.Report(ctx, err)
	return nil
}
