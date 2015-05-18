package trace_test

import (
	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/trace"
	"golang.org/x/net/context"
)

func ping(ctx context.Context) (err error) {
	ctx, done := trace.Trace(ctx)
	defer done(nil, "pong")

	return
}

func Example() {
	ctx := context.Background()
	ctx = logger.WithLogger(ctx, logger.Stdout)
	ping(ctx)

	// pong trace.id=<uuid>
	// trace.func=github.com/remind101/pkg/trace_test.ping trace.file=... trace.line=10
}
