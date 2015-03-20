// package hb is a Go package from sending errors to Honeybadger.
package hb

import (
	"fmt"
	"runtime"
	"strings"

	"golang.org/x/net/context"
)

var DefaultMax = 1024

// Reporter is used to report errors to honeybadger.
type Reporter struct {
	// Generate is a function used to generate a Report.
	Generator Generator

	// http client to use when sending reports to honeybadger.
	client interface {
		Send(*Report) error
	}
}

// NewReporter returns a new Reporter instance.
func NewReporter(key string, g Generator) *Reporter {
	return &Reporter{
		Generator: g,
		client: &Client{
			Key: key,
		},
	}
}

// Report reports the error to honeybadger.
func (r *Reporter) Report(ctx context.Context, err error) error {
	g := r.Generator
	if g == nil {
		g = NewReportGenerator("dev")
	}

	report, err2 := g.Generate(ctx, err)
	if err2 != nil {
		return err2
	}

	return r.client.Send(report)
}

func fullMessage(msg interface{}) string {
	return fmt.Sprintf("%s", msg)
}

func exceptionClass(message string) string {
	pieces := strings.Split(message, ":")
	for i := len(pieces) - 1; i >= 0; i-- {
		err := strings.TrimSpace(pieces[i])
		if len(err) > 0 {
			return err
		}
	}
	return ""
}

func functionName(pc uintptr) string {
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		return "???"
	}
	return fn.Name()
}
