package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

func TestRequestID(t *testing.T) {
	tests := []struct {
		header http.Header
		id     string
	}{
		{http.Header{http.CanonicalHeaderKey("X-Request-ID"): []string{"1234"}}, "1234"},
		{http.Header{http.CanonicalHeaderKey("Request-ID"): []string{"1234"}}, "1234"},
		{http.Header{http.CanonicalHeaderKey("Foo"): []string{"1234"}}, ""},
	}

	for _, tt := range tests {
		m := &RequestID{
			handler: httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
				requestID := httpx.RequestID(ctx)

				if got, want := requestID, tt.id; got != want {
					t.Fatalf("RequestID => %s; want %s", got, want)
				}

				return nil
			}),
		}

		ctx := context.Background()
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/", nil)
		req.Header = tt.header

		if err := m.ServeHTTPContext(ctx, resp, req); err != nil {
			t.Fatal(err)
		}
	}
}
