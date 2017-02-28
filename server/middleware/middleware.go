package middleware

import (
	"net/http"

	"golang.org/x/net/context"

	"github.com/inconshreveable/log15"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/logger"
)

// Handler adapts an httpx.Handler to an http.Handler using the provided root
// context.
func Handler(root context.Context, h httpx.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTPContext(root, w, r)
	})
}

// Common wraps the handler with common middleware to:
//
// * Log requests
// * Recover from panics.
// * Add the request id to the context.
func Common(h httpx.Handler) httpx.Handler {
	// Log requests to the embedded logger.
	h = LogRequests(h)

	// Prefix log messages with the request id.
	h = PrefixRequestID(h)

	// Add information about the request to reported errors.
	return WithRequest(h)
}

// LogRequests logs the requests to the embedded logger.
func LogRequests(h httpx.Handler) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		logger.Info(ctx, "request.start",
			"method", r.Method,
			"path", r.URL.Path,
		)

		err := h.ServeHTTPContext(ctx, w, r)

		logger.Info(ctx, "request.complete")

		return err
	})
}

// PrefixRequestID adds the request as a prefix to the log15.Logger.
func PrefixRequestID(h httpx.Handler) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		if l, ok := logger.FromContext(ctx); ok {
			if l, ok := l.(log15.Logger); ok {
				ctx = logger.WithLogger(ctx, l.New("request_id", httpx.RequestID(ctx)))
			}
		}

		return h.ServeHTTPContext(ctx, w, r)
	})
}
