// package httpx provides an extra layer of convenience over package http.
package httpx

import (
	"net/http"

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

// key used to store context values from within this package.
type key int

const (
	varsKey key = iota
	requestIDKey
	routeKey
)
