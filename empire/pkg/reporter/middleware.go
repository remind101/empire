package reporter

import (
	"fmt"
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

	// ErrorHandler is an httpx.Handler that will be called when a panic
	// occurs.
	ErrorHandler httpx.Handler

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

	defer func() {
		if v := recover(); v != nil {
			err := fmt.Errorf("%v", v)

			if v, ok := v.(error); ok {
				err = v
			}

			h.Report(ctx, err)

			if h.ErrorHandler != nil {
				h.ErrorHandler.ServeHTTPContext(ctx, w, r)
			}
		}
	}()

	err := h.handler.ServeHTTPContext(ctx, w, r)
	if err != nil {
		h.Report(ctx, err)
	}

	return err
}
