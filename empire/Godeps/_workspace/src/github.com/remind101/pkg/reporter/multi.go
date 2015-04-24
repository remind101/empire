package reporter

import "golang.org/x/net/context"

// MultiReporter is an implementation of the Reporter interface that reports the
// error to multiple Reporters. If any of the individual error reporters returns
// an error, a MutliError will be returned.
type MultiReporter []Reporter

func (r MultiReporter) Report(ctx context.Context, err error) error {
	var errors []error

	for _, reporter := range r {
		if err2 := reporter.Report(ctx, err); err2 != nil {
			errors = append(errors, err2)
		}
	}

	if len(errors) == 0 {
		return nil
	}

	return &MultiError{Errors: errors}
}
