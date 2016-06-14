package middleware

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

func TestRecovery(t *testing.T) {
	var (
		called  bool
		errBoom = errors.New("boom")
	)

	ctx := reporter.WithReporter(context.Background(), reporter.ReporterFunc(func(ctx context.Context, err error) error {
		called = true

		e := err.(*reporter.Error)

		if e.Err != errBoom {
			t.Fatalf("err => %v; want %v", err, errBoom)
		}

		return nil
	}))

	h := &Recovery{
		handler: httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			panic(errBoom)
		}),
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.Header.Set("X-Request-ID", "1234")
	resp := httptest.NewRecorder()

	ctx = httpx.WithRequest(ctx, req)

	defer func() {
		if err := recover(); err != nil {
			t.Fatal("Expected the panic to be handled.")
		}
	}()

	err := h.ServeHTTPContext(ctx, resp, req)

	if err != errBoom {
		t.Fatalf("err => %v; want %v", err, errBoom)
	}
}

func TestRecoveryPanicString(t *testing.T) {
	ctx := reporter.WithReporter(context.Background(), reporter.ReporterFunc(func(ctx context.Context, err error) error {
		return nil
	}))

	h := &Recovery{
		handler: httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			panic("boom")
		}),
	}

	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

	defer func() {
		if err := recover(); err != nil {
			t.Fatal("Expected the panic to be handled.")
		}
	}()

	err := h.ServeHTTPContext(ctx, resp, req)

	if got, want := err.Error(), "boom"; got != want {
		t.Fatalf("err => %v; want %v", got, want)
	}
}
