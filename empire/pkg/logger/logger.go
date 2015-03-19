// package logger is a package that provides a structured logger that's
// context.Context aware.
package logger

import (
	"fmt"
	"io"
	"log"
	"sort"
	"strings"

	"github.com/remind101/empire/empire/pkg/httpx"
	"golang.org/x/net/context"
)

// Logger represents a structured logger.
type Logger interface {
	Log(map[string]interface{})
}

// logger is an implementation of the Logger interface.
type logger struct {
	*log.Logger
	prefix map[string]interface{}
}

// Log logs the pairs in logfmt.
func (l *logger) Log(pairs map[string]interface{}) {
	var (
		prefix []string
		parts  []string
	)

	for k, v := range l.prefix {
		prefix = append(prefix, fmt.Sprintf("%s=%v", k, v))
	}

	for k, v := range pairs {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	sort.Strings(parts)

	l.Println(strings.Join(append(prefix, parts...), " "))
}

// New returns a new log.Logger with the request id as the log prefix.
func New(r io.Writer, requestID httpx.RequestID) Logger {
	return &logger{
		Logger: log.New(r, "", 0),
		prefix: map[string]interface{}{
			"request_id": requestID,
		},
	}
}

// WithLogger inserts a log.Logger into the provided context.
func WithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns a log.Logger from the context.
func FromContext(ctx context.Context) (Logger, bool) {
	l, ok := ctx.Value(loggerKey).(Logger)
	return l, ok
}

// Log is a convenience method, which extracts a logger from the context object,
// then calls the Log method on it.
func Log(ctx context.Context, pairs map[string]interface{}) {
	if l, ok := FromContext(ctx); ok {
		l.Log(pairs)
	}
}

type key int

const (
	loggerKey key = iota
)
