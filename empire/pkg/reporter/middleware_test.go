package reporter

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

func TestMiddleware(t *testing.T) {
	var called bool

	r := ReporterFunc(func(ctx context.Context, err error) error {
		called = true

		if err != ErrFake {
			t.Fatalf("err => %v; want %v", err, ErrFake)
		}

		return nil
	})

	h := httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		_, ok := FromContext(ctx)
		if !ok {
			t.Fatal("Expected the handler to be added to the context")
		}

		return ErrFake
	})
	m := &Middleware{
		Reporter: r,
		handler:  h,
	}

	ctx := context.Background()
	req, _ := http.NewRequest("GET", "/", nil)
	resp := httptest.NewRecorder()

	err := m.ServeHTTPContext(ctx, resp, req)

	if err != ErrFake {
		t.Fatalf("err => %v; want %v", err, ErrFake)
	}

	if !called {
		t.Fatal("Expected reporter to be called")
	}
}
