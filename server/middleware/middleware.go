package middleware

import (
	"fmt"
	"hash/fnv"
	"net/http"

	"golang.org/x/net/context"

	"github.com/inconshreveable/log15"
	"github.com/remind101/empire/tracer"
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

	// Recover from panics by reporting them to the reporter.
	h = WithRecovery(h)

	// Add a root span to the request.
	h = WithTracing(h)

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

// WithTracing adds a root trace to the request.
func WithTracing(h httpx.Handler) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		span := tracer.NewRootSpanFromContext(ctx, "http.request", "server", fmt.Sprintf("%s Unknown", r.Method))
		span.Type = "http"
		span.SetMeta("http.method", r.Method)
		span.SetMeta("http.url", r.URL.String())

		if id := httpx.RequestID(ctx); id != "" {
			span.SetMeta("http.request_id", id)
			span.TraceID = traceID(id)
			span.SpanID = span.TraceID
		}

		err := h.ServeHTTPContext(context.WithValue(span.Context(ctx), rootSpanKey, span), w, r)
		span.FinishWithErr(err)
		return err
	})
}

// traceID returns a new traceID as a 64 bit hash of the request ID, so that you
// can deterministically find a trace, given a request id.
func traceID(requestID string) uint64 {
	if requestID == "" {
		// Programmer error
		panic("no request id")
	}
	h := fnv.New64()
	h.Write([]byte(requestID))
	return h.Sum64()
}

// Returns the root span embeded from the top level request.
func RootSpan(ctx context.Context) *tracer.Span {
	if ctx == nil {
		return &tracer.Span{}
	}
	span, ok := ctx.Value(rootSpanKey).(*tracer.Span)
	if !ok {
		return &tracer.Span{}
	}
	return span
}

type key int

const (
	rootSpanKey key = iota
)
