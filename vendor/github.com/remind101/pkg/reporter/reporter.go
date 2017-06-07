// package reporter provides a context.Context aware abstraction for shuttling
// errors and panics to third partys.
package reporter

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context"
)

// DefaultMax is the default maximum number of lines to show from the stack trace.
var DefaultMax = 1024

// Reporter represents an error handler.
type Reporter interface {
	// Report reports the error to an external system. The provided error
	// could be an Error instance, which will contain additional information
	// about the error, including a stack trace and any contextual
	// information. Implementers should type assert the error to an *Error
	// if they want to report the stack trace.
	Report(context.Context, error) error
}

// ReporterFunc is a function signature that conforms to the Reporter interface.
type ReporterFunc func(context.Context, error) error

// Report implements the Reporter interface.
func (f ReporterFunc) Report(ctx context.Context, err error) error {
	return f(ctx, err)
}

// FromContext extracts a Reporter from a context.Context.
func FromContext(ctx context.Context) (Reporter, bool) {
	h, ok := ctx.Value(reporterKey).(Reporter)
	return h, ok
}

// WithReporter inserts a Reporter into the context.Context.
func WithReporter(ctx context.Context, r Reporter) context.Context {
	return context.WithValue(withInfo(ctx), reporterKey, r)
}

// AddContext adds contextual information to the Request object.
func AddContext(ctx context.Context, key string, value interface{}) {
	i := infoFromContext(ctx)
	i.context[key] = value
}

// AddRequest adds information from an http.Request to the Request object.
func AddRequest(ctx context.Context, req *http.Request) {
	i := infoFromContext(ctx)
	// TODO clone the request?
	i.request = req
}

// newError returns a new Error instance. If err is already an Error instance,
// it will be returned, otherwise err will be wrapped with NewErrorWithContext.
func newError(ctx context.Context, err error) *Error {
	if e, ok := err.(*Error); ok {
		return e
	} else {
		return NewErrorWithContext(ctx, err, 2)
	}
}

// Report wraps the err as an Error and reports it the the Reporter embedded
// within the context.Context.
func Report(ctx context.Context, err error) error {
	e := newError(ctx, err)

	if r, ok := FromContext(ctx); ok {
		return r.Report(ctx, e)
	} else {
		panic("No reporter in provided context.")
	}

	return nil
}

// Monitors and reports panics. Useful in goroutines.
// Example:
//   ctx := reporter.WithReporter(context.Background(), hb2.NewReporter(hb2.Config{}))
//   ...
//   go func(ctx context.Context) {
//     defer reporter.Monitor(ctx)
//     ...
//     panic("oh noes") // will report, then crash.
//   }(ctx)
func Monitor(ctx context.Context) {
	if v := recover(); v != nil {
		var err error
		if e, ok := v.(error); ok {
			err = e
		} else {
			err = fmt.Errorf("panic: %v", v)
		}
		Report(ctx, err)
		panic(err)
	}
}

// Error wraps an error with additional information, like a stack trace,
// contextual information, and an http request if provided.
type Error struct {
	// The error that was generated.
	Err error

	// Any freeform contextual information about that error.
	Context map[string]interface{}

	// If provided, an http request that generated the error.
	Request *http.Request

	// This is private so that it can be exposed via StackTrace(),
	// which implements the stackTracker interface.
	stackTrace errors.StackTrace
}

// Make Error implement the error interface.
func (e *Error) Error() string {
	return e.Err.Error()
}

// Make Error implement the causer interface.
func (e *Error) Cause() error {
	return errors.Cause(e.Err)
}

// Make Error implement the stackTracer interface.
func (e *Error) StackTrace() errors.StackTrace {
	return e.stackTrace
}

// NewError wraps err as an Error and generates a stack trace pointing at the
// caller of this function.
func NewError(err error, skip int) *Error {
	return &Error{
		Err:        err,
		stackTrace: stacktrace(err, skip+1),
	}
}

// NewErrorWithContext returns a new Error with contextual information added.
func NewErrorWithContext(ctx context.Context, err error, skip int) *Error {
	e := NewError(err, skip+1)
	i := infoFromContext(ctx)
	e.Context = i.context
	e.Request = i.request
	return e
}

// MutliError is an error implementation that wraps multiple errors.
type MultiError struct {
	Errors []error
}

// Error implements the error interface. It simply joins all of the individual
// error messages with a comma.
func (e *MultiError) Error() string {
	var m []string

	for _, err := range e.Errors {
		m = append(m, err.Error())
	}

	return strings.Join(m, ", ")
}

type causer interface {
	Cause() error
}

type stackTracer interface {
	StackTrace() errors.StackTrace
}

// It generates a brand new stack trace given an error and
// the number of frames that should be skipped,
// from innermost to outermost frames.
func gen_stacktrace(err error, skip int) errors.StackTrace {
	var stack errors.StackTrace
	err_with_stack := errors.WithStack(err)
	stack = err_with_stack.(stackTracer).StackTrace()
	skip += 1

	// if it is recovering from a panic() call,
	// reset the stack trace at that point
	for index, frame := range stack {
		file := fmt.Sprintf("%s", frame)
		if file == "panic.go" {
			skip = index + 1
			break
		}
	}

	return stack[skip:]
}

// There are two interfaces that drive this implementation:
//
//   * causer
//     - it unwraps an error instance in a chain of errors created with errors.Wrap
//     - therefore, the last one in the chain is the root cause (inner-most)
//
//   * stackTracer
//     - not all errors in the aforementioned chain may have a stack trace,
//
// It returns the innermost stack trace in a chain of errors because it is
// the closest to the root cause.
//
func get_stacktrace(err error) errors.StackTrace {
	var stack errors.StackTrace
	for err != nil {
		err_with_stack, stack_ok := err.(stackTracer)
		if stack_ok && err_with_stack.StackTrace() != nil {
			stack = err_with_stack.StackTrace()
		}
		if err_with_cause, causer_ok := err.(causer); causer_ok {
			err = err_with_cause.Cause()
		} else {
			// end of chain
			break
		}
	}
	return stack
}

func stacktrace(err error, skip int) errors.StackTrace {
	stack := get_stacktrace(err)
	if stack == nil {
		stack = gen_stacktrace(err, skip+1)
	}
	if len(stack) > DefaultMax {
		stack = stack[:DefaultMax]
	}
	return stack
}

// info is used internally to store contextual information. Any empty info
// gets inserted into the context.Context when the Reporter is inserted, which
// allows downstream consumers to add additional information to this object.
type info struct {
	context map[string]interface{}
	request *http.Request
}

func newInfo() *info {
	return &info{context: make(map[string]interface{})}
}

func withInfo(ctx context.Context) context.Context {
	return context.WithValue(ctx, infoKey, newInfo())
}

func infoFromContext(ctx context.Context) *info {
	i, ok := ctx.Value(infoKey).(*info)
	if !ok {
		return newInfo()
	}
	return i
}

// key used to store context values from within this package.
type key int

const (
	reporterKey key = iota
	infoKey
)
