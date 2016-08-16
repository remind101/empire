package middleware

import (
	"context"
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/remind101/pkg/logger"
)

// WithContext will fall back to the given context.Context for any values not
// provided in the request's context.Context.
func WithContext(h http.Handler, ctx context.Context) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r = r.WithContext(&fallbackContext{
			Context:  r.Context(),
			fallback: ctx,
		})
		h.ServeHTTP(w, r)
	})
}

type fallbackContext struct {
	context.Context
	fallback context.Context
}

func (c *fallbackContext) Value(key interface{}) interface{} {
	if v := c.Context.Value(key); v != nil {
		return v
	}

	return c.fallback.Value(key)
}

// Common wraps the handler with common middleware to:
//
// * Log requests
// * Add the request id to the context.
func Common(h http.Handler) http.Handler {
	// Log requests to the embedded logger.
	h = LogRequests(h)

	// Prefix log messages with the request id.
	h = PrefixRequestID(h)

	// Add information about the request to reported errors.
	return WithRequest(h)
}

// LogRequests logs the requests to the embedded logger.
func LogRequests(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		logger.Info(ctx, "request.start",
			"method", r.Method,
			"path", r.URL.Path,
		)

		h.ServeHTTP(w, r)

		logger.Info(ctx, "request.complete")
	})
}

// PrefixRequestID adds the request as a prefix to the log15.Logger.
func PrefixRequestID(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		if l, ok := logger.FromContext(ctx); ok {
			if l, ok := l.(log15.Logger); ok {
				ctx = logger.WithLogger(ctx, l.New("request_id", requestID(r)))
				r = r.WithContext(ctx)
			}
		}

		h.ServeHTTP(w, r)
	})
}

func requestID(r *http.Request) string {
	return r.Header.Get(http.CanonicalHeaderKey("X-Request-Id"))
}
