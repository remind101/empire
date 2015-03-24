package middleware

import (
	"net/http"
	"os"

	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/httpx/middleware"
	"github.com/remind101/empire/empire/pkg/reporter"
)

type CommonOpts struct {
	// A Reporter to use to report errors and panics.
	Reporter reporter.Reporter

	// An ErrorHandler to respond with a pretty error message.
	ErrorHandler func(error, http.ResponseWriter, *http.Request)
}

// Common wraps the httpx.Handler with some common middleware.
func Common(h httpx.Handler, opts CommonOpts) http.Handler {
	errorHandler := opts.ErrorHandler

	// Wrap the router in middleware to handle errors.
	h = middleware.HandleError(h, errorHandler)

	// Recover from panics.
	h = middleware.Recover(h, opts.Reporter)

	// The recovered panic should be pretty too.
	h = middleware.HandleError(h, errorHandler)

	// Add a logger to the context.
	h = middleware.NewLogger(h, os.Stdout)

	// Add the request id to the context.
	h = middleware.ExtractRequestID(h)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}
