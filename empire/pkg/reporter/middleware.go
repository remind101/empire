package reporter

import (
	"net/http"

	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

// Middleware is an httpx.Handler that sets the error handler in the
// context.Context object and also
type Middleware struct {
	// Reporter is a Reporter that will be inserted into the context. It
	// will also be used to report panics.
	Reporter

	// Handler is the wrapped httpx.Handler to call.
	handler httpx.Handler
}

func NewMiddleware(h httpx.Handler, r Reporter) *Middleware {
	return &Middleware{
		Reporter: r,
		handler:  h,
	}
}

// ServeHTTP implements the http.Handler interface.
func (h *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.ServeHTTPContext(context.Background(), w, r)
}

// ServeHTTPContext implements the httpx.Handler interface.
func (h *Middleware) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx = WithReporter(ctx, h.Reporter)

	err := h.handler.ServeHTTPContext(ctx, w, r)
	if err != nil {
		h.Report(ctx, err)
	}

	return err
}
