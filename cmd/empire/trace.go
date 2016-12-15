package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/remind101/empire/tracer"
	"github.com/remind101/pkg/logger"
)

// tracerTransport is a trace.Transport implementation that will log, and
// optionally forward traces to another transport.
type tracerTransport struct {
	http   tracer.Transport
	Logger logger.Logger
}

// newTracerTransport returns a new tracer.Tracer instance that sends traces to
// the given host, and also logs the traces if l is not nil.
func newTracer(hostname, port string, l logger.Logger) *tracer.Tracer {
	if port == "" {
		port = "7777"
	}
	if hostname == "" {
		hostname = "localhost"
	}
	transport := &tracerTransport{
		http:   tracer.NewHTTPTransport(fmt.Sprintf("http://%s:%s/v0.3/traces", hostname, port)),
		Logger: l,
	}
	return tracer.NewTracerTransport(transport)
}

func (t *tracerTransport) Send(traces [][]*tracer.Span) (*http.Response, error) {
	if t.Logger != nil {
		for _, group := range traces {
			for _, trace := range group {
				t.Logger.Debug(
					"trace",
					"trace_id", trace.TraceID,
					"span_id", trace.SpanID,
					"parent_id", trace.ParentID,
					"service", trace.Service,
					"name", trace.Name,
					"resource", trace.Resource,
					"duration", time.Duration(trace.Duration),
				)
			}
		}
	}

	if t.http != nil {
		return t.http.Send(traces)
	}

	return nil, nil
}

func (t *tracerTransport) SetHeader(key, value string) {
	if t.http != nil {
		t.http.SetHeader(key, value)
	}
}
