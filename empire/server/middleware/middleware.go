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
	h1 := middleware.HandleError(h, errorHandler)

	// Recover from panics.
	h2 := middleware.Recover(h1)

	// Report the panics to the reporter.
	h3 := middleware.Report(h2, opts.Reporter)

	// The recovered panic should be pretty too.
	h4 := middleware.HandleError(h3, errorHandler)

	// Add a logger to the context.
	h5 := middleware.NewLogger(h4, os.Stdout)

	// Add the request id to the context.
	h6 := middleware.ExtractRequestID(h5)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h6)
}
