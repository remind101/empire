package middleware

import (
	"fmt"
	"net/http"

	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

// Recovery is a middleware that will recover from panics and return the error.
type Recovery struct {
	// handler is the wrapped httpx.Handler.
	handler httpx.Handler
}

func Recover(h httpx.Handler) *Recovery {
	return &Recovery{
		handler: h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface. It recovers from
// panics and returns an error for upstream middleware to handle.
func (h *Recovery) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) (err error) {
	defer func() {
		if v := recover(); v != nil {
			err = fmt.Errorf("%v", v)

			if v, ok := v.(error); ok {
				err = v
			}

			return
		}
	}()

	err = h.handler.ServeHTTPContext(ctx, w, r)

	return
}
