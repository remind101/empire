package tracer

import (
	"log"
	"time"

	"context"
)

const (
	flushInterval = 2 * time.Second
)

// Tracer creates, buffers and submits Spans which are used to time blocks of
// compuration.
//
// When a tracer is disabled, it will not submit spans for processing.
type Tracer struct {
	transport Transport // is the transport mechanism used to delivery spans to the agent
	sampler   sampler   // is the trace sampler to only keep some samples

	buffer *spansBuffer

	DebugLoggingEnabled bool
	enabled             bool // defines if the Tracer is enabled or not
}

// NewTracer creates a new Tracer. Most users should use the package's
// DefaultTracer instance.
func NewTracer() *Tracer {
	return NewTracerTransport(newDefaultTransport())
}

// NewTracerTransport create a new Tracer with the given transport.
func NewTracerTransport(transport Transport) *Tracer {
	t := &Tracer{
		enabled:             true,
		transport:           transport,
		buffer:              newSpansBuffer(spanBufferDefaultMaxSize),
		sampler:             newAllSampler(),
		DebugLoggingEnabled: false,
	}

	// start a background worker
	go t.worker()

	return t
}

// SetEnabled will enable or disable the tracer.
func (t *Tracer) SetEnabled(enabled bool) {
	t.enabled = enabled
}

// Enabled returns whether or not a tracer is enabled.
func (t *Tracer) Enabled() bool {
	return t.enabled
}

// SetSampleRate sets a sample rate for all the future traces.
// sampleRate has to be between 0 (sample nothing) and 1 (sample everything).
func (t *Tracer) SetSampleRate(sampleRate float64) {
	if sampleRate == 1 {
		t.sampler = newAllSampler()
	} else if sampleRate >= 0 && sampleRate < 1 {
		t.sampler = newRateSampler(sampleRate)
	} else {
		log.Printf("tracer.SetSampleRate rate must be between 0 and 1, now: %f", sampleRate)
	}
}

// NewRootSpan creates a span with no parent. Its ids will be randomly
// assigned.
func (t *Tracer) NewRootSpan(name, service, resource string) *Span {
	spanID := nextSpanID()
	span := NewSpan(name, service, resource, spanID, spanID, 0, t)
	t.sampler.Sample(span)
	return span
}

// NewChildSpan returns a new span that is child of the Span passed as
// argument.
func (t *Tracer) NewChildSpan(name string, parent *Span) *Span {
	spanID := nextSpanID()

	// when we're using parenting in inner functions, it's possible that
	// a nil pointer is sent to this function as argument. To prevent a crash,
	// it's better to be defensive and to produce a wrongly configured span
	// that is not sent to the trace agent.
	if parent == nil {
		span := NewSpan(name, "", name, spanID, spanID, spanID, t)
		t.sampler.Sample(span)
		return span
	}

	// child that is correctly configured
	span := NewSpan(name, parent.Service, name, spanID, parent.TraceID, parent.SpanID, parent.tracer)
	// child sampling same as the parent
	span.Sampled = parent.Sampled

	return span
}

// NewChildSpanFromContext will create a child span of the span contained in
// the given context. If the context contains no span, an empty span will be
// returned.
func (t *Tracer) NewChildSpanFromContext(name string, ctx context.Context) *Span {
	span, _ := SpanFromContext(ctx) // tolerate nil spans
	return t.NewChildSpan(name, span)
}

// record queues the finished span for further processing.
func (t *Tracer) record(span *Span) {
	if t.enabled && span.Sampled {
		t.buffer.Push(span)
	}
}

// Flush will push any currently buffered traces to the server.
func (t *Tracer) Flush() error {
	spans := t.buffer.Pop()

	if t.DebugLoggingEnabled {
		log.Printf("Sending %d spans", len(spans))
		for _, s := range spans {
			log.Printf("SPAN:\n%s", s.String())
		}
	}

	// bal if there's nothing to do
	if !t.enabled || t.transport == nil || len(spans) == 0 {
		return nil
	}

	// rebuild the traces list; this operation is done in the Flush() instead
	// after each record() because this avoids a huge number of initializations
	// and RW mutex locks, keeping the same performance as before (except for this
	// little overhead). The overall optimization (and idiomatic code) could be
	// reached replacing all our buffers with channels.
	var traces [][]*Span
	traceBuffer := make(map[uint64][]*Span)
	for _, s := range spans {
		traceBuffer[s.TraceID] = append(traceBuffer[s.TraceID], s)
	}
	for _, t := range traceBuffer {
		traces = append(traces, t)
	}

	_, err := t.transport.Send(traces)
	return err
}

// worker periodically flushes traces to the transport.
func (t *Tracer) worker() {
	for range time.Tick(flushInterval) {
		err := t.Flush()
		if err != nil {
			log.Printf("[WORKER] flush failed, lost spans: %s", err)
		}
	}
}

// DefaultTracer is a global tracer that is enabled by default. All of the
// packages top level NewSpan functions use the default tracer.
//
//	span := tracer.NewRootSpan("sql.query", "user-db", "select * from foo where id = ?")
//	defer span.Finish()
//
var DefaultTracer = NewTracer()

// NewRootSpan creates a span with no parent. It's ids will be randomly
// assigned.
func NewRootSpan(name, service, resource string) *Span {
	return DefaultTracer.NewRootSpan(name, service, resource)
}

// NewChildSpan creates a span that is a child of parent. It will inherit the
// parent's service and resource.
func NewChildSpan(name string, parent *Span) *Span {
	return DefaultTracer.NewChildSpan(name, parent)
}

// NewChildSpanFromContext will create a child span of the span contained in
// the given context. If the context contains no span, a span with
// no service or resource will be returned.
func NewChildSpanFromContext(name string, ctx context.Context) *Span {
	return DefaultTracer.NewChildSpanFromContext(name, ctx)
}

// Enable will enable the default tracer.
func Enable() {
	DefaultTracer.SetEnabled(true)
}

// Disable will disable the default tracer.
func Disable() {
	DefaultTracer.SetEnabled(false)
}
