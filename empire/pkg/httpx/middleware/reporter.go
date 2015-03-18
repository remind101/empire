package middleware

import (
	"net/http"

	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/reporter"
	"golang.org/x/net/context"
)

// Reporter is a middleware that will report errors to a reporter.Reporter
type Reporter struct {
	// Reporter is a Reporter that will be inserted into the context. It
	// will also be used to report panics.
	reporter.Reporter

	// Handler is the wrapped httpx.Handler to call.
	handler httpx.Handler
}

func Report(h httpx.Handler, r reporter.Reporter) *Reporter {
	return &Reporter{
		Reporter: r,
		handler:  h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface.
func (h *Reporter) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	ctx = reporter.WithReporter(ctx, h.Reporter)

	err := h.handler.ServeHTTPContext(ctx, w, r)
	if err != nil {
		h.Report(ctx, err)
	}

	return err
}
