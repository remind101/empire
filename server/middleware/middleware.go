package middleware

import (
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

var Background = middleware.BackgroundContext

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

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}
