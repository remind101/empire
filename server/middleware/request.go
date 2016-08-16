package middleware

import (
	"net/http"

	"github.com/remind101/pkg/reporter"
)

// WithRequest adds information about the http.Request to reported errors.
func WithRequest(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Add the request to the context.
		reporter.AddRequest(ctx, r)

		// Add the request id
		reporter.AddContext(ctx, "request_id", requestID(r))

		h.ServeHTTP(w, r)
	})
}
