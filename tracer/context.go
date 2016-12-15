package tracer

import "context"

type datadogContextKey struct{}

var (
	spanKey   = datadogContextKey{}
	tracerKey = datadogContextKey{}
)

// NewRootSpan returns a new root Span using the tracer embeded in the Context.
func NewRootSpanFromContext(ctx context.Context, name, service, resource string) *Span {
	tracer, ok := TracerFromContext(ctx)
	if !ok {
		return &Span{}
	}
	return tracer.NewRootSpan(name, service, resource)
}

// TracerFromContext returns the embeded Tracer from the Context, if provided.
func TracerFromContext(ctx context.Context) (*Tracer, bool) {
	if ctx == nil {
		return nil, false
	}
	tracer, ok := ctx.Value(tracerKey).(*Tracer)
	return tracer, ok
}

// WithTracer embeds the given tracer in the context.
func WithTracer(ctx context.Context, tracer *Tracer) context.Context {
	if tracer == nil {
		return ctx
	}
	return context.WithValue(ctx, tracerKey, tracer)
}

// ContextWithSpan will return a new context that includes the given span.
// DEPRECATED: use span.Context(ctx) instead.
func ContextWithSpan(ctx context.Context, span *Span) context.Context {
	if span == nil {
		return ctx
	}
	return span.Context(ctx)
}

// SpanFromContext returns the stored *Span from the Context if it's available.
// This helper returns also the ok value that is true if the span is present.
func SpanFromContext(ctx context.Context) (*Span, bool) {
	if ctx == nil {
		return nil, false
	}
	span, ok := ctx.Value(spanKey).(*Span)
	return span, ok
}

// SpanFromContextDefault returns the stored *Span from the Context. If not, it
// will return an empty span that will do nothing.
func SpanFromContextDefault(ctx context.Context) *Span {

	// FIXME[matt] is it better to return a singleton empty span?
	if ctx == nil {
		return &Span{}
	}

	span, ok := SpanFromContext(ctx)
	if !ok {
		return &Span{}
	}
	return span
}
