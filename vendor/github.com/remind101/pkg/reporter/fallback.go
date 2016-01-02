package reporter

import "golang.org/x/net/context"

type FallbackReporter struct {
	// The first reporter to call.
	Reporter Reporter

	// This reporter will be used to report an error if the first Reporter
	// fails for some reason.
	Fallback Reporter
}

func (r *FallbackReporter) Report(ctx context.Context, err error) error {
	if err2 := r.Reporter.Report(ctx, err); err2 != nil {
		r.Fallback.Report(ctx, err2)
		return err2
	}

	return nil
}
