package httpx

import (
	"net/http"

	"golang.org/x/net/context"
)

// RequestID extracts a RequestID from a context.
func RequestID(ctx context.Context) string {
	requestID, _ := ctx.Value("http.request.id").(string)
	return requestID
}

// headerExtractor returns a function that can extract a request id from a list
// of headers.
func headerExtractor(headers ...string) func(*http.Request) string {
	return func(r *http.Request) string {
		for _, h := range headers {
			v := r.Header.Get(h)
			if v != "" {
				return v
			}
		}

		return ""
	}
}
