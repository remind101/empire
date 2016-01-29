package middleware

import (
	"fmt"
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
	"golang.org/x/net/trace"
)

type CommonOpts struct {
	// A Reporter to use to report errors and panics.
	Reporter reporter.Reporter

	// A logger to log requests to.
	Logger log15.Logger
}

type log struct {
	log15.Logger
}

func (l *log) New(pairs ...interface{}) logger.Logger {
	return &log{l.Logger.New(pairs...)}
}

// Common wraps the httpx.Handler with some common middleware.
func Common(h httpx.Handler, opts CommonOpts) http.Handler {
	l := &log{opts.Logger}

	// Recover from panics.
	h = middleware.Recover(h, opts.Reporter)

	// Add a logger to the context.
	h = middleware.LogTo(h, func(ctx context.Context, r *http.Request) logger.Logger {
		return l.New("request_id", httpx.RequestID(ctx))
	})

	// Insert a trace.Trace for tracing requests.
	h = withTracing(h)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}

// withTracing wraps an httpx.Handler to insert a trace.Trace into the context.
func withTracing(h httpx.Handler) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		tr := trace.New("http.request", fmt.Sprintf("%s %s", r.Method, r.URL.String()))
		defer tr.Finish()
		return h.ServeHTTPContext(trace.NewContext(ctx, tr), w, r)
	})
}
