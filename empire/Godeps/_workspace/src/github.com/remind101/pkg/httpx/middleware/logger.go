package middleware

import (
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

// Logger is middleware that will insert a logger.Logger into the context.
type Logger struct {
	// Device is an io.Writer to write logs to.
	Device io.Writer

	// handler is the wrapped httpx.Handler
	handler httpx.Handler
}

func NewLogger(h httpx.Handler, d io.Writer) *Logger {
	return &Logger{
		Device:  d,
		handler: h,
	}
}

func (h *Logger) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	l := logger.New(log.New(h.Device, fmt.Sprintf("request_id=%s ", httpx.RequestID(ctx)), 0))
	ctx = logger.WithLogger(ctx, l)

	l.Log(
		"at", "request",
		"method", r.Method,
		"path", fmt.Sprintf(`"%s"`, r.URL.Path),
	)

	return h.handler.ServeHTTPContext(ctx, w, r)
}
