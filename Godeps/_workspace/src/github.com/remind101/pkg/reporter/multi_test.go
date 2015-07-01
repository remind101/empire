package reporter

import (
	"errors"
	"testing"

	"golang.org/x/net/context"
)

func TestMultiReporter(t *testing.T) {
	var (
		r1Called bool
		r2Called bool
	)

	r1 := ReporterFunc(func(ctx context.Context, err error) error {
		r1Called = true
		return nil
	})

	r2 := ReporterFunc(func(ctx context.Context, err error) error {
		r2Called = true
		return nil
	})

	h := MultiReporter{r1, r2}

	if err := h.Report(context.Background(), errBoom); err != nil {
		t.Fatal(err)
	}

	if got, want := r1Called, true; got != want {
		t.Fatal("Expected r1 to be called")
	}

	if got, want := r2Called, true; got != want {
		t.Fatal("Expected r2 to be called")
	}
}

// Tests when the Report method of the individual reporters returns an error.
func TestMultiReporterError(t *testing.T) {
	r1 := ReporterFunc(func(ctx context.Context, err error) error {
		return errors.New("boom 1")
	})

	r2 := ReporterFunc(func(ctx context.Context, err error) error {
		return errors.New("boom 2")
	})

	h := MultiReporter{r1, r2}

	err := h.Report(context.Background(), errBoom)

	if _, ok := err.(*MultiError); !ok {
		t.Fatal("Expected a MultiError to be returned")
	}
}
