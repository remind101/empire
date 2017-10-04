package middleware

import (
	"net/http"

	"github.com/inconshreveable/log15"
	"github.com/remind101/empire/internal/realip"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/logger"
)

// Common wraps the handler with common middleware to:
//
// * Log requests
// * Recover from panics.
// * Add the request id to the context.
func Common(h http.Handler, r *realip.Resolver) http.Handler {
	// Log requests to the embedded logger.
	h = LogRequests(h)

	// Prefix log messages with the request id.
	h = PrefixRequestID(h)

	// Parse out the real ip address of the request.
	h = realip.Middleware(h, r)

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
			"user_agent", r.Header.Get("User-Agent"),
			"remote_ip", realip.RealIP(r),
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
				ctx = logger.WithLogger(ctx, l.New("request_id", httpx.RequestID(ctx)))
			}
		}

		h.ServeHTTP(w, r.WithContext(ctx))
	})
}
