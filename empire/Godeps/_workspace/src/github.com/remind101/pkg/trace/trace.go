package trace

import (
	"runtime"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/remind101/pkg/logger"
	"github.com/remind101/pkg/reporter"
	"golang.org/x/net/context"
)

// newID is used internally to generate a trace id.
var newID = uuid.New

func Trace(ctx context.Context) (context.Context, func(error, string, ...interface{})) {
	pc, file, line, _ := runtime.Caller(1)
	f := runtime.FuncForPC(pc)
	ctx = &tracedContext{
		Context: ctx,
		id:      newID(),
		parent:  traceID(ctx),
		start:   time.Now(),
		fnname:  f.Name(),
		file:    file,
		line:    line,
	}

	return ctx, func(err error, msg string, pairs ...interface{}) {
		l, ok := logger.WithValues(ctx, "trace.id", "trace.parent", "trace.func", "trace.file", "trace.line", "trace.duration")
		if ok {
			l.Info(msg, pairs...)
		}

		// Report the error to the reporter.
		reporter.ReportWithSkip(ctx, err, 1)
	}
}

// tracedContext is a context.Context implementation that provides information
// about a trace.
type tracedContext struct {
	context.Context
	start  time.Time
	id     string
	parent string
	fnname string
	file   string
	line   int
}

func (ctx *tracedContext) Value(v interface{}) interface{} {
	if key, ok := v.(string); ok {
		switch key {
		case "trace.id":
			return ctx.id
		case "trace.parent":
			return ctx.parent
		case "trace.func":
			return ctx.fnname
		case "trace.file":
			return ctx.file
		case "trace.line":
			return ctx.line
		case "trace.duration":
			return time.Since(ctx.start)
		}
	}

	return ctx.Context.Value(v)
}

func traceID(ctx context.Context) string {
	id, _ := ctx.Value("trace.id").(string)
	return id
}
