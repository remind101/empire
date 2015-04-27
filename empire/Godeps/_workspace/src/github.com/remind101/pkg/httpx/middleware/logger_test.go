// Thanks negroni!
package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

func TestLogger(t *testing.T) {
	b := new(bytes.Buffer)

	h := LogTo(httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(201)
		return nil
	}), stdLogger(b))

	ctx := context.Background()
	resp := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/", nil)

	if err := h.ServeHTTPContext(ctx, resp, req); err != nil {
		t.Fatal(err)
	}

	if got, want := b.String(), "request_id= request.start method=GET path=/\nrequest_id= request.complete status=201\n"; got != want {
		t.Fatalf("%s; want %s", got, want)
	}
}
