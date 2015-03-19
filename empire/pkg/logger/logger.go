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
	Log(pairs ...interface{})
}

// logger is an implementation of the Logger interface.
type logger struct {
	*log.Logger
	prefix map[string]interface{}
}

// Log logs the pairs in logfmt. It will treat consecutive arguments as a key
// value pair. Given the input:
func (l *logger) Log(pairs ...interface{}) {
	p := l.prefixMessage()
	m := l.message(pairs...)

	// No message, so we just print the prefix to avoid printing an extra
	// space at the end.
	if m == "" {
		l.Println(p)
		return
	}

	l.Println(fmt.Sprintf("%s %s", p, m))
}

func (l *logger) prefixMessage() string {
	var parts []string

	for k, v := range l.prefix {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	sort.Strings(parts)

	return strings.Join(parts, " ")
}

func (l *logger) message(pairs ...interface{}) string {
	if len(pairs) == 1 {
		return fmt.Sprintf("%v", pairs[0])
	}

	var parts []string

	for i := 0; i < len(pairs); i += 2 {
		// This conditional means that the pairs are uneven and we've
		// reached the end of iteration. We treat the last value as a
		// simple string message. Given an input pair as:
		//
		//	["key", "value", "message"]
		//
		// The output will be:
		//
		//	key=value message
		if len(pairs) == i+1 {
			parts = append(parts, fmt.Sprintf("%v", pairs[i]))
		} else {
			parts = append(parts, fmt.Sprintf("%s=%v", pairs[i], pairs[i+1]))
		}
	}

	return strings.Join(parts, " ")
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
func Log(ctx context.Context, pairs ...interface{}) {
	if l, ok := FromContext(ctx); ok {
		l.Log(pairs...)
	}
}

type key int

const (
	loggerKey key = iota
)
