// Package trace wraps net/trace for convenience.
package trace

import (
	"fmt"
	"runtime"

	"golang.org/x/net/context"
	"golang.org/x/net/trace"
)

// Location prints the current file name, function and line number.
func Location(ctx context.Context) {
	if tr, ok := trace.FromContext(ctx); ok {
		pc, file, line, _ := runtime.Caller(1)
		fn := runtime.FuncForPC(pc)
		tr.LazyPrintf("%s.%s:%d", file, fn.Name(), line)
	}
}

func Log(ctx context.Context, v interface{}, sensitive bool) {
	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyLog(&fmtStringer{v: v}, sensitive)
	}
}

func LazyPrintf(ctx context.Context, fmt string, v ...interface{}) {
	if tr, ok := trace.FromContext(ctx); ok {
		tr.LazyPrintf(fmt, v...)
	}
}

func SetError(ctx context.Context, err error) {
	if err != nil {
		if tr, ok := trace.FromContext(ctx); ok {
			LazyPrintf(ctx, "error: %v", err)
			tr.SetError()
		}
	}
}

type fmtStringer struct {
	v interface{}
}

func (s *fmtStringer) String() string {
	return fmt.Sprintf("%#v", s.v)
}
