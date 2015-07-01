package nr

import (
	"errors"
	"testing"

	"github.com/remind101/newrelic"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

var (
	// boom
	errBoom = errors.New("boom")

	// boom with backtrace.
	errBoomMore = reporter.NewError(errBoom, 0)
)

func TestReport(t *testing.T) {
	tx := newrelic.NewTx("GET /boom")
	tx.Reporter = &TestReporter{
		f: func(id int64, exceptionType, errorMessage, stackTrace, stackFrameDelim string) {
			if got, want := exceptionType, "*errors.errorString"; got != want {
				t.Errorf("exceptionType => %v; want %v", got, want)
			}
			if got, want := errorMessage, "boom"; got != want {
				t.Errorf("errorMessage => %v; want %v", got, want)
			}

			if stackTrace == "" {
				t.Error("stackTrace: expected to not be empty")
			}
		},
	}

	ctx := context.Background()
	ctx = newrelic.WithTx(ctx, tx)

	r := NewReporter()
	r.Report(ctx, errBoomMore)
}

type TestReporter struct {
	f func(id int64, exceptionType, errorMessage, stackTrace, stackFrameDelim string)
}

func (r *TestReporter) ReportError(id int64, exceptionType, errorMessage, stackTrace, stackFrameDelim string) (int, error) {
	r.f(id, exceptionType, errorMessage, stackTrace, stackFrameDelim)
	return 0, nil
}
