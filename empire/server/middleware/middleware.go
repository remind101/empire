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

type CommonOpts struct {
	// A Reporter to use to report errors and panics.
	Reporter reporter.Reporter

	// A logger to log requests to.
	Logger log15.Logger
}

// Common wraps the httpx.Handler with some common middleware.
func Common(h httpx.Handler, opts CommonOpts) http.Handler {
	// Recover from panics.
	h = middleware.Recover(h, opts.Reporter)

	// Add a logger to the context.
	h = middleware.LogTo(h, func(ctx context.Context, r *http.Request) logger.Logger {
		return opts.Logger.New("request_id", httpx.RequestID(ctx))
	})

	// Add the request id to the context.
	h = middleware.ExtractRequestID(h)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}
