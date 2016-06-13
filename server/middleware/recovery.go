package middleware

import (
	"fmt"
	"net/http"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

// Recovery is a middleware that will recover from panics and return the error.
type Recovery struct {
	// handler is the wrapped httpx.Handler.
	handler httpx.Handler
}

func WithRecovery(h httpx.Handler) *Recovery {
	return &Recovery{
		handler: h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface. It recovers from
// panics and returns an error for upstream middleware to handle.
func (h *Recovery) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
	if _, ok := reporter.FromContext(ctx); ok {
		defer func() {
			if v := recover(); v != nil {
				err = fmt.Errorf("%v", v)

				if v, ok := v.(error); ok {
					err = v
				}

				reporter.Report(ctx, err)

				return
			}
		}()
	}

	err = h.handler.ServeHTTPContext(ctx, w, r)

	return
}
