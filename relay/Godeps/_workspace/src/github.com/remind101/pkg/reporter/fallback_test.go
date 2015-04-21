package reporter

import (
	"errors"
	"testing"

	"golang.org/x/net/context"
)

func TestFallback(t *testing.T) {
	var called bool

	errTimeout := errors.New("net: timeout")

	r := &FallbackReporter{
		Reporter: ReporterFunc(func(ctx context.Context, err error) error {
			return errTimeout
		}),
		Fallback: ReporterFunc(func(ctx context.Context, err error) error {
			called = true

			if got, want := err, errTimeout; got != want {
				t.Fatalf("err => %v; want %v", got, want)
			}

			return nil
		}),
	}

	r.Report(context.Background(), errBoom)

	if !called {
		t.Fatal("fallback not called")
	}
}
