// package hb is a Go package from sending errors to Honeybadger.
package hb

import (
	"fmt"
	"os"
	"strings"

	"github.com/remind101/pkg/reporter"
	"github.com/remind101/pkg/reporter/util"
	"golang.org/x/net/context"
)

// Ensure that Reporter implements the reporter.Reporter interface.
var _ reporter.Reporter = &Reporter{}

// Headers that won't be sent to honeybadger.
var IgnoredHeaders = map[string]struct{}{
	"Authorization": struct{}{},
}

// Reporter is used to report errors to honeybadger.
type Reporter struct {
	Environment string

	// http client to use when sending reports to honeybadger.
	client interface {
		Send(*Report) error
	}

	cwd      string
	hostname string
}

// NewReporter returns a new Reporter instance.
func NewReporter(key string) *Reporter {
	cwd, _ := os.Getwd()
	hostname, _ := os.Hostname()

	return &Reporter{
		client:   NewClientFromKey(key),
		hostname: hostname,
		cwd:      cwd,
	}
}

// Report reports the error to honeybadger.
func (r *Reporter) Report(ctx context.Context, err error) error {
	report := r.NewReport(err)
	return r.client.Send(report)
}

func (r *Reporter) NewReport(err error) *Report {
	report := NewReport(err)
	report.Server.ProjectRoot["path"] = r.cwd
	report.Server.EnvironmentName = r.Environment
	report.Server.Hostname = r.hostname

	return report
}

// NewReport generates a new honeybadger report from an error.
func NewReport(err error) *Report {
	r := &Report{
		Notifier: &Notifier{
			Name:     "Honeybadger (Go)",
			Url:      "https://github.com/remind101/pkg/reporter/hb",
			Version:  "0.1",
			Language: "Go",
		},
		Error: &Error{
			Class:     util.ClassName(err),
			Message:   err.Error(),
			Backtrace: make([]*BacktraceLine, 0),
			Source:    make(map[string]interface{}),
		},
		Request: &Request{
			Params:  make(map[string]interface{}),
			Session: make(map[string]interface{}),
			CgiData: make(map[string]interface{}),
			Context: make(map[string]interface{}),
		},
		Server: &Server{
			ProjectRoot: make(map[string]interface{}),
		},
	}

	if e, ok := err.(*reporter.Error); ok {
		r.Error.Class = util.ClassName(e.Err)

		for _, l := range e.Backtrace {
			r.Error.Backtrace = append(r.Error.Backtrace, &BacktraceLine{
				Method: util.FunctionName(l.PC),
				File:   l.File,
				Number: fmt.Sprintf("%d", l.Line),
			})
		}

		if e.Request != nil {
			r.Request.Url = e.Request.URL.String()

			for header, values := range e.Request.Header {
				if _, ok := IgnoredHeaders[header]; ok {
					continue
				}

				h := strings.Replace(strings.ToUpper(header), "-", "_", -1)
				r.Request.CgiData["HTTP_"+h] = values
			}

			r.Request.CgiData["REQUEST_METHOD"] = e.Request.Method
		}

		for key, value := range e.Context {
			r.Request.Context[key] = value
		}
	}

	return r
}
