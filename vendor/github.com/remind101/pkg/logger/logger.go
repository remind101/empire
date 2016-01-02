// package logger is a package that provides a structured logger that's
// context.Context aware.
package logger

import (
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/net/context"
)

var Stdout = New(log.New(os.Stdout, "", 0))

// Logger represents a structured leveled logger.
type Logger interface {
	Debug(msg string, pairs ...interface{})
	Info(msg string, pairs ...interface{})
	Warn(msg string, pairs ...interface{})
	Error(msg string, pairs ...interface{})
	Crit(msg string, pairs ...interface{})
}

var _ childLogging = &logger{}

// childLogging is an interface that loggers can implement to support
// child/prefixed logging.
type childLogging interface {
	New(pairs ...interface{}) Logger
}

// logger is an implementation of the Logger interface backed by the stdlib's
// logging facility. This is a fairly naive implementation, and it's probably
// better to use something like https://github.com/inconshreveable/log15 which
// offers real structure logging.
type logger struct {
	*log.Logger
}

// New wraps the log.Logger to implement the Logger interface.
func New(l *log.Logger) Logger {
	return &logger{
		Logger: l,
	}
}

// New implemens the childLogging interface.
func (l *logger) New(pairs ...interface{}) Logger {
	return l
}

// Log logs the pairs in logfmt. It will treat consecutive arguments as a key
// value pair. Given the input:
func (l *logger) Log(msg string, pairs ...interface{}) {
	m := l.message(pairs...)
	l.Println(msg, m)
}

func (l *logger) Debug(msg string, pairs ...interface{}) { l.Log(msg, pairs...) }
func (l *logger) Info(msg string, pairs ...interface{})  { l.Log(msg, pairs...) }
func (l *logger) Warn(msg string, pairs ...interface{})  { l.Log(msg, pairs...) }
func (l *logger) Error(msg string, pairs ...interface{}) { l.Log(msg, pairs...) }
func (l *logger) Crit(msg string, pairs ...interface{})  { l.Log(msg, pairs...) }

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

// WithLogger inserts a log.Logger into the provided context.
func WithLogger(ctx context.Context, l Logger) context.Context {
	return context.WithValue(ctx, loggerKey, l)
}

// FromContext returns a log.Logger from the context.
func FromContext(ctx context.Context) (Logger, bool) {
	l, ok := ctx.Value(loggerKey).(Logger)
	return l, ok
}

// WithValues returns a new logger prefixed with the values of the given keys
// after being extracted from the context.
func WithValues(ctx context.Context, keys ...string) (Logger, bool) {
	l, ok := FromContext(ctx)
	if !ok {
		return l, ok
	}

	if l, ok := l.(childLogging); ok {
		return l.New(contextPairs(ctx, keys...)...), true
	}

	// TODO: Return false if the logger doesn't support child logging?
	return l, true
}

func Info(ctx context.Context, msg string, pairs ...interface{}) {
	withLogger(ctx, func(l Logger) {
		l.Info(msg, pairs...)
	})
}

func InfoContext(ctx context.Context, msg string, keys ...string) {
	Info(ctx, msg, contextPairs(ctx, keys...)...)
}

func Debug(ctx context.Context, msg string, pairs ...interface{}) {
	withLogger(ctx, func(l Logger) {
		l.Debug(msg, pairs...)
	})
}

func Warn(ctx context.Context, msg string, pairs ...interface{}) {
	withLogger(ctx, func(l Logger) {
		l.Warn(msg, pairs...)
	})
}

func Error(ctx context.Context, msg string, pairs ...interface{}) {
	withLogger(ctx, func(l Logger) {
		l.Error(msg, pairs...)
	})
}

func Crit(ctx context.Context, msg string, pairs ...interface{}) {
	withLogger(ctx, func(l Logger) {
		l.Crit(msg, pairs...)
	})
}

func withLogger(ctx context.Context, fn func(l Logger)) {
	if l, ok := FromContext(ctx); ok {
		fn(l)
	}
}

// contextPairs takes a slice of string keys, obtains their values from the
// context.Context and returns the suitable list of key value pairs as a
// []interface{}.
func contextPairs(ctx context.Context, keys ...string) []interface{} {
	var pairs []interface{}
	for _, k := range keys {
		pairs = append(pairs, k, ctx.Value(k))
	}
	return pairs
}

type key int

const (
	loggerKey key = iota
)
