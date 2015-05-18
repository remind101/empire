package middleware

import (
	"net/http"

	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

// DefaultGenerator is the default context generator. Defaults to just use
// context.Background().
var DefaultGenerator = context.Background

// Background is middleware that implements the http.Handler interface to inject
// an initial context object. Use this as the entry point from an http.Handler
// server.
type Background struct {
	// Generate will be called to generate a context.Context for the
	// request.
	Generate func() context.Context

	// The wrapped httpx.Handler to call down to.
	handler httpx.Handler
}

func BackgroundContext(h httpx.Handler) *Background {
	return &Background{
		handler: h,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *Background) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := h.Generate
	if ctx == nil {
		ctx = DefaultGenerator
	}
	h.ServeHTTPContext(httpx.WithRequest(ctx(), r), w, r)
}

func (h *Background) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return h.handler.ServeHTTPContext(ctx, w, r)
}
