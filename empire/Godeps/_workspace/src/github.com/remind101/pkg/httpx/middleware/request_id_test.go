package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

func TestRequestID(t *testing.T) {
	m := &RequestID{
		handler: httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			requestID := httpx.RequestID(ctx)

			if got, want := requestID, "1234"; got != want {
				t.Fatalf("RequestID => %s; want %s", got, want)
			}

			return nil
		}),
	}

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "1234")

	if err := m.ServeHTTPContext(ctx, resp, req); err != nil {
		t.Fatal(err)
	}
}
