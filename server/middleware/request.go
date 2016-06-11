package middleware

import (
	"net/http"

	"golang.org/x/net/context"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/reporter"
)

// WithRequest adds information about the http.Request to reported errors.
func WithRequest(h httpx.Handler) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		ctx = httpx.WithRequest(ctx, r)

		// Add the request to the context.
		reporter.AddRequest(ctx, r)

		// Add the request id
		reporter.AddContext(ctx, "request_id", httpx.RequestID(ctx))

		return h.ServeHTTPContext(ctx, w, r)
	})
}
