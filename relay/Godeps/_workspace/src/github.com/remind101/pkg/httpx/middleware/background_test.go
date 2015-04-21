package middleware

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

func TestBackground(t *testing.T) {
	m := &Background{
		handler: httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			if ctx == nil {
				t.Fatal("Expected a context to be generated")
			}

			io.WriteString(w, `Ok`)
			return nil
		}),
	}

	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

	m.ServeHTTP(resp, req)

	if got, want := resp.Body.String(), `Ok`; got != want {
		t.Fatalf("Body => %s; want %s", got, want)
	}
}
