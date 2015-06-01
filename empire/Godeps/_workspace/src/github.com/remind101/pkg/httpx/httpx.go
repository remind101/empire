// package httpx provides an extra layer of convenience over package http.
package httpx

import (
	"net/http"
	"time"

	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

// Handler is represents a Handler that can take a context.Context as the
// first argument.
type Handler interface {
	ServeHTTPContext(context.Context, http.ResponseWriter, *http.Request) error
}

// The HandlerFunc type is an adapter to allow the use of ordinary functions as
// httpx handlers. If f is a function with the appropriate signature,
// HandlerFunc(f) is a Handler object that calls f.
type HandlerFunc func(context.Context, http.ResponseWriter, *http.Request) error

// ServeHTTPContext calls f(ctx, w, r)
func (f HandlerFunc) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return f(ctx, w, r)
}

// requestContext is a context.Context implementation that provides details
// about an http request.
type requestContext struct {
	context.Context

	startedAt time.Time
	r         *http.Request
}

// WithRequest inserts an http.Request into the context.
func WithRequest(ctx context.Context, r *http.Request) context.Context {
	return &requestContext{
		Context:   ctx,
		startedAt: timex.Now(),
		r:         r,
	}
}

// RequestFromContext extracts the http.Request from the context.Context.
func RequestFromContext(ctx context.Context) (*http.Request, bool) {
	r, ok := ctx.Value(requestKey).(*http.Request)
	return r, ok
}

// Value implements the Value method of the context.Context interface. If the
// provided key is a requestKey, the raw http.Request will be returned. You can
// also provided string keys prefixed with "http.request." to get values from
// the http.Request object.
func (ctx *requestContext) Value(v interface{}) interface{} {
	if k, ok := v.(key); ok && k == requestKey {
		return ctx.r
	}

	if key, ok := v.(string); ok {
		switch key {
		case "http.request.method":
			return ctx.r.Method
		case "http.request.id":
			f := headerExtractor("X-Request-Id", "Request-Id")
			return f(ctx.r)
		case "http.request.uri":
			return ctx.r.RequestURI
		case "http.request.startedat":
			return ctx.startedAt
		}
	}

	return ctx.Context.Value(v)
}

// key used to store context values from within this package.
type key int

const (
	varsKey key = iota
	requestKey
	routeKey
)
