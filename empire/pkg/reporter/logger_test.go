package reporter

import (
	"bytes"
	"testing"

	"github.com/remind101/empire/empire/pkg/logger"
	"golang.org/x/net/context"
)

func TestLogReporter(t *testing.T) {
	b := new(bytes.Buffer)
	l := logger.New(b, "1234")
	h := &LogReporter{}

	ctx := logger.WithLogger(context.Background(), l)
	if err := h.Report(ctx, ErrFake); err != nil {
		t.Fatal(err)
	}

	if got, want := b.String(), "request_id=1234 error=\"boom\"\n"; got != want {
		t.Fatalf("Output => %s; want %s", got, want)
	}
}
