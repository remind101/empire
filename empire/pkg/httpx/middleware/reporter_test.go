package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/reporter"
	"golang.org/x/net/context"
)

func TestReporter(t *testing.T) {
	var (
		called  bool
		errBoom = errors.New("boom")
	)

	r := reporter.ReporterFunc(func(ctx context.Context, err error) error {
		called = true

		if err != errBoom {
			t.Fatalf("err => %v; want %v", err, errBoom)
		}

		return nil
	})

	h := httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		_, ok := reporter.FromContext(ctx)
		if !ok {
			t.Fatal("Expected the handler to be added to the context")
		}

		return errBoom
	})
	m := &Reporter{
		Reporter: r,
		handler:  h,
	}

	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

	err := m.ServeHTTPContext(ctx, resp, req)

	if err != errBoom {
		t.Fatalf("err => %v; want %v", err, errBoom)
	}

	if !called {
		t.Fatal("Expected reporter to be called")
	}
}
