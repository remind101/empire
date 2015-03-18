package reporter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

func TestMiddleware(t *testing.T) {
	r := ReporterFunc(func(ctx context.Context, err error) error {
		return nil
	})
	h := httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		_, ok := FromContext(ctx)
		if !ok {
			t.Fatal("Expected the handler to be added to the context")
		}

		return nil
	})
	m := &Middleware{
		Reporter: r,
		handler:  h,
	}

	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

	if err := m.ServeHTTPContext(ctx, resp, req); err != nil {
		t.Fatal(err)
	}
}

func TestMiddlewarePanic(t *testing.T) {
	var called bool

	r := ReporterFunc(func(ctx context.Context, err error) error {
		called = true
		return nil
	})
	h := httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		panic(ErrFake)
		return nil
	})
	m := &Middleware{
		Reporter: r,
		handler:  h,
	}

	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

	defer func() {
		if err := recover(); err != nil {
			t.Fatal("Expected the panic to be handled.")
		}
	}()

	if err := m.ServeHTTPContext(ctx, resp, req); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("Expected the error to be handled")
	}
}

func TestMiddlewarePanicString(t *testing.T) {
	var called bool

	r := ReporterFunc(func(ctx context.Context, err error) error {
		called = true

		if got, want := err.Error(), "fuck"; got != want {
			t.Fatalf("Error => %s; want %s", got, want)
		}

		return nil
	})
	h := httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		panic("fuck")
		return nil
	})
	m := &Middleware{
		Reporter: r,
		handler:  h,
	}

	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

	defer func() {
		if err := recover(); err != nil {
			t.Fatal("Expected the panic to be handled.")
		}
	}()

	if err := m.ServeHTTPContext(ctx, resp, req); err != nil {
		t.Fatal(err)
	}

	if !called {
		t.Fatal("Expected the error to be handled")
	}
}
