package middleware

import (
	"net/http"
	"os"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
	"github.com/remind101/pkg/reporter"
)

type CommonOpts struct {
	// A Reporter to use to report errors and panics.
	Reporter reporter.Reporter
}

// Common wraps the httpx.Handler with some common middleware.
func Common(h httpx.Handler, opts CommonOpts) http.Handler {
	// Recover from panics.
	h = middleware.Recover(h, opts.Reporter)

	// Add a logger to the context.
	h = middleware.NewLogger(h, os.Stdout)

	// Add the request id to the context.
	h = middleware.ExtractRequestID(h)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}
