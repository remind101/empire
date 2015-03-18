package reporter

import (
	"bytes"
	"log"
	"testing"

	"golang.org/x/net/context"
)

func TestLogReporter(t *testing.T) {
	b := new(bytes.Buffer)
	l := log.New(b, "", 0)
	h := &LogReporter{Logger: l}

	if err := h.Report(context.Background(), ErrFake); err != nil {
		t.Fatal(err)
	}

	if got, want := b.String(), "boom\n"; got != want {
		t.Fatalf("Output => %s; want %s", got, want)
	}
}
