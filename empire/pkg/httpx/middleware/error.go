package middleware

import (
	"net/http"

	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

// DefaultErrorHandler is an error handler that will respond with the error
// message and a 500 status.
var DefaultErrorHandler = func(err error, w http.ResponseWriter, r *http.Request) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// Error is an httpx.Handler that will handle errors with an ErrorHandler.
type Error struct {
	// ErrorHandler is a function that will be called when a handler returns
	// an error.
	ErrorHandler func(error, http.ResponseWriter, *http.Request)

	// Handler is the wrapped httpx.Handler that will be called.
	handler httpx.Handler
}

func NewError(h httpx.Handler) *Error {
	return &Error{
		handler: h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface.
func (h *Error) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	err := h.handler.ServeHTTPContext(ctx, w, r)

	if err != nil {
		f := h.ErrorHandler
		if f == nil {
			f = DefaultErrorHandler
		}

		f(err, w, r)
	}

	// Bubble up the error to any upstream middleware.
	return err
}
