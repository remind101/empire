package middleware

import (
	"net/http"

	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

// HeaderRequestID is the default name of the header to extract request ids
// from.
const HeaderRequestID = "X-Request-Id"

// RequestID is middleware that extracts a request id from the headers and
// inserts it into the context.
type RequestID struct {
	// Header is the name of the http header to extract the request id from.
	// The zero value is the value of HeaderRequestID.
	Header string

	// handler is the wrapped httpx.Handler.
	handler httpx.Handler
}

func ExtractRequestID(h httpx.Handler) *RequestID {
	return &RequestID{
		handler: h,
	}
}

// ServeHTTPContext implements the httpx.Handler interface. It extracts a
// request id from the headers and inserts it into the context.
func (h *RequestID) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	requestID := extractRequestID(r, h.Header)

	ctx = httpx.WithRequestID(ctx, requestID)
	return h.handler.ServeHTTPContext(ctx, w, r)
}

func extractRequestID(r *http.Request, header string) string {
	if header == "" {
		header = HeaderRequestID
	}

	return r.Header.Get(header)
}
