package hb

import (
	"fmt"
	"os"
	"reflect"
	"runtime"

	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

// Generator represents an object that can generate a Report.
type Generator interface {
	Generate(context.Context, error) (*Report, error)
}

type ReportGenerator struct {
	// The honeybadger environment.
	Environment string

	// Maximum number of lines to show in the backtrace. The zero value is
	// DefaultMax.
	Max int

	// Skip controls the number of callers to skip.
	Skip int

	cwd      string
	hostname string
}

// NewReportGenerator returns a new ReportGenerator instance.
func NewReportGenerator(env string) *ReportGenerator {
	cwd, _ := os.Getwd()
	hostname, _ := os.Hostname()

	return &ReportGenerator{
		Environment: env,
		cwd:         cwd,
		hostname:    hostname,
	}
}

func (g *ReportGenerator) Generate(ctx context.Context, err error) (*Report, error) {
	r := &Report{
		Notifier: &Notifier{
			Name:     "Honeybadger (Go)",
			Url:      "https://github.com/remind101/empire/pkg/reporter/hb",
			Version:  "0.1",
			Language: "Go",
		},
		Error: &Error{
			Class:     reflect.TypeOf(err).String(),
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
			ProjectRoot: map[string]interface{}{
				"path": g.cwd,
			},
			EnvironmentName: g.Environment,
			Hostname:        g.hostname,
		},
	}

	start := 1 + g.Skip

	max := g.Max
	if max == 0 {
		max = DefaultMax
	}

	max = max + g.Skip

	for i := start; i < max; i++ {
		pc, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}

		r.Error.Backtrace = append(r.Error.Backtrace, &BacktraceLine{
			Method: functionName(pc),
			File:   file,
			Number: fmt.Sprintf("%d", line),
		})
	}

	return r, nil
}

// RequestIDGenerator is a Generator implementation that wraps another Generator
// to add the RequestID to the Report.
type RequestIDGenerator struct {
	Generator
}

func AddRequestID(g Generator) Generator {
	return &RequestIDGenerator{
		Generator: g,
	}
}

func (g *RequestIDGenerator) Generate(ctx context.Context, err error) (*Report, error) {
	report, err2 := g.Generator.Generate(ctx, err)
	if err2 != nil {
		return report, err2
	}

	report.AddContext("request_id", httpx.RequestID(ctx))

	return report, nil
}
