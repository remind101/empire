package nr

import (
	"fmt"
	"strings"

	"github.com/remind101/newrelic"
	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/util"
	"golang.org/x/net/context"
)

// Ensure that Reporter implements the reporter.Reporter interface.
var _ reporter.Reporter = &Reporter{}

type Reporter struct{}

func NewReporter() *Reporter {
	return &Reporter{}
}

func (r *Reporter) Report(ctx context.Context, err error) error {
	if tx, ok := newrelic.FromContext(ctx); ok {
		var (
			exceptionType   string
			errorMessage    string
			stackTrace      []string
			stackFrameDelim string
		)

		errorMessage = err.Error()
		stackFrameDelim = "\n"
		stackTrace = make([]string, 0)

		if e, ok := err.(*reporter.Error); ok {
			exceptionType = util.ClassName(e.Err)

			for _, l := range e.Backtrace {
				stackTrace = append(stackTrace, fmt.Sprintf("%s:%d %s", l.File, l.Line, util.FunctionName(l.PC)))
			}

		}

		return tx.ReportError(exceptionType, errorMessage, strings.Join(stackTrace, stackFrameDelim), stackFrameDelim)
	}
	return nil
}
