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

	h := &Recovery{
		Reporter: reporter.ReporterFunc(func(ctx context.Context, err error) error {
			called = true

			if err != errBoom {
				t.Fatalf("err => %v; want %v", err, errBoom)
			}

			return nil
		}),
		handler: httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			panic(errBoom)
		}),
	}

	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

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
	h := &Recovery{
		Reporter: reporter.ReporterFunc(func(ctx context.Context, err error) error {
			return nil
		}),
		handler: httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
			panic("boom")
		}),
	}

	ctx := context.Background()
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
