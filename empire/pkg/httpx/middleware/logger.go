package middleware

import (
	"fmt"
	"io"
	"net/http"

	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/pkg/logger"
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
	l := logger.New(h.Device, httpx.RequestIDFromContext(ctx))
	ctx = logger.WithLogger(ctx, l)

	l.Log(map[string]interface{}{
		"at":     "request",
		"method": r.Method,
		"path":   fmt.Sprintf(`"%s"`, r.URL.Path),
	})

	return h.handler.ServeHTTPContext(ctx, w, r)
}
