package reporter

import (
	"bytes"
	"log"
	"testing"

	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

func TestLogReporter(t *testing.T) {
	tests := []struct {
		err error
		out string
	}{
		{errBoom, "request_id=1234  error=\"boom\"\n"},
		{&Error{Err: errBoom}, "request_id=1234  error=\"boom\" line=0 file=unknown\n"},
		{&Error{Err: errBoom, Backtrace: []*BacktraceLine{&BacktraceLine{File: "foo.go", Line: 1}}}, "request_id=1234  error=\"boom\" line=1 file=foo.go\n"},
	}

	for i, tt := range tests {
		b := new(bytes.Buffer)
		l := logger.New(log.New(b, "request_id=1234 ", 0))
		h := &LogReporter{}

		ctx := logger.WithLogger(context.Background(), l)
		if err := h.Report(ctx, tt.err); err != nil {
			t.Fatal(err)
		}

		if got, want := b.String(), tt.out; got != want {
			t.Fatalf("#%d: Output => %s; want %s", i, got, want)
		}
	}
}
