package middleware

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

// StdoutLogger is a logger.Logger generator that generates a logger that writes
// to stdout.
var StdoutLogger = stdLogger(os.Stdout)

// LogTo is an httpx middleware that wraps the handler to insert a logger and
// log the request to it.
func LogTo(h httpx.Handler, f func(context.Context, *http.Request) logger.Logger) httpx.Handler {
	return InsertLogger(Log(h), f)
}

// InsertLogger returns an httpx.Handler middleware that will call f to generate
// a logger, then insert it into the context.
func InsertLogger(h httpx.Handler, f func(context.Context, *http.Request) logger.Logger) httpx.Handler {
	return httpx.HandlerFunc(func(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
		l := f(ctx, r)
		ctx = logger.WithLogger(ctx, l)
		return h.ServeHTTPContext(ctx, w, r)
	})
}

func stdLogger(out io.Writer) func(context.Context, *http.Request) logger.Logger {
	return func(ctx context.Context, r *http.Request) logger.Logger {
		return logger.New(log.New(out, fmt.Sprintf("request_id=%s ", httpx.RequestID(ctx)), 0))
	}
}

// Logger is middleware that logs the request details to the logger.Logger
// embedded within the context.
type Logger struct {
	// handler is the wrapped httpx.Handler
	handler httpx.Handler
}

func Log(h httpx.Handler) *Logger {
	return &Logger{
		handler: h,
	}
}

func (h *Logger) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	rw := NewResponseWriter(w)

	logger.Info(ctx, "request.start",
		"method", r.Method,
		"path", r.URL.Path,
	)

	err := h.handler.ServeHTTPContext(ctx, rw, r)

	logger.Info(ctx, "request.complete",
		"status", rw.Status(),
	)

	return err
}
