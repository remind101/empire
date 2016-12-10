package tracer

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultTracer(t *testing.T) {
	assert := assert.New(t)

	// the default client must be available
	assert.NotNil(DefaultTracer)

	// package free functions must proxy the calls to the
	// default client
	root := NewRootSpan("pylons.request", "pylons", "/")
	NewChildSpan("pylons.request", root)
	Disable()
	Enable()
}

func TestNewSpan(t *testing.T) {
	assert := assert.New(t)

	// the tracer must create root spans
	tracer := NewTracer()
	span := tracer.NewRootSpan("pylons.request", "pylons", "/")
	assert.Equal(span.ParentID, uint64(0))
	assert.Equal(span.Service, "pylons")
	assert.Equal(span.Name, "pylons.request")
	assert.Equal(span.Resource, "/")
}

func TestNewSpanFromContextNil(t *testing.T) {
	assert := assert.New(t)
	tracer := NewTracer()

	child := tracer.NewChildSpanFromContext("abc", nil)
	assert.Equal(child.Name, "abc")
	assert.Equal(child.Service, "")

	child = tracer.NewChildSpanFromContext("def", context.Background())
	assert.Equal(child.Name, "def")
	assert.Equal(child.Service, "")

}

func TestNewSpanFromContext(t *testing.T) {
	assert := assert.New(t)

	// the tracer must create child spans
	tracer := NewTracer()
	parent := tracer.NewRootSpan("pylons.request", "pylons", "/")
	ctx := ContextWithSpan(context.Background(), parent)

	child := tracer.NewChildSpanFromContext("redis.command", ctx)
	// ids and services are inherited
	assert.Equal(child.ParentID, parent.SpanID)
	assert.Equal(child.TraceID, parent.TraceID)
	assert.Equal(child.Service, parent.Service)
	// the resource is not inherited and defaults to the name
	assert.Equal(child.Resource, "redis.command")
	// the tracer instance is the same
	assert.Equal(parent.tracer, tracer)
	assert.Equal(child.tracer, tracer)

}

func TestNewSpanChild(t *testing.T) {
	assert := assert.New(t)

	// the tracer must create child spans
	tracer := NewTracer()
	parent := tracer.NewRootSpan("pylons.request", "pylons", "/")
	child := tracer.NewChildSpan("redis.command", parent)
	// ids and services are inherited
	assert.Equal(child.ParentID, parent.SpanID)
	assert.Equal(child.TraceID, parent.TraceID)
	assert.Equal(child.Service, parent.Service)
	// the resource is not inherited and defaults to the name
	assert.Equal(child.Resource, "redis.command")
	// the tracer instance is the same
	assert.Equal(parent.tracer, tracer)
	assert.Equal(child.tracer, tracer)
}

func TestTracerDisabled(t *testing.T) {
	assert := assert.New(t)

	// disable the tracer and be sure that the span is not added
	tracer := NewTracer()
	tracer.SetEnabled(false)
	span := tracer.NewRootSpan("pylons.request", "pylons", "/")
	span.Finish()
	assert.Equal(tracer.buffer.Len(), 0)
}

func TestTracerEnabledAgain(t *testing.T) {
	assert := assert.New(t)

	// disable the tracer and enable it again
	tracer := NewTracer()
	tracer.SetEnabled(false)
	preSpan := tracer.NewRootSpan("pylons.request", "pylons", "/")
	preSpan.Finish()
	tracer.SetEnabled(true)
	postSpan := tracer.NewRootSpan("pylons.request", "pylons", "/")
	postSpan.Finish()
	assert.Equal(tracer.buffer.Len(), 1)
}

func TestTracerSampler(t *testing.T) {
	assert := assert.New(t)

	sampleRate := 0.5
	tracer := NewTracer()
	tracer.SetSampleRate(sampleRate)

	span := tracer.NewRootSpan("pylons.request", "pylons", "/")

	// The span might be sampled or not, we don't know, but at least it should have the sample rate metric
	assert.Equal(sampleRate, span.Metrics[sampleRateMetricKey])
}

func TestTracerEdgeSampler(t *testing.T) {
	assert := assert.New(t)

	// a sample rate of 0 should sample nothing
	tracer0 := NewTracer()
	tracer0.SetSampleRate(0)
	// a sample rate of 1 should sample everything
	tracer1 := NewTracer()
	tracer1.SetSampleRate(1)

	count := 10000

	for i := 0; i < count; i++ {
		span0 := tracer0.NewRootSpan("pylons.request", "pylons", "/")
		span0.Finish()
		span1 := tracer1.NewRootSpan("pylons.request", "pylons", "/")
		span1.Finish()
	}

	assert.Equal(0, tracer0.buffer.Len())
	assert.Equal(count, tracer1.buffer.Len())
}

func TestTracerConcurrent(t *testing.T) {
	assert := assert.New(t)
	tracer, transport := getTestTracer()

	// Wait for three different goroutines that should create
	// three different traces with one child each
	var wg sync.WaitGroup
	wg.Add(3)
	go func() {
		defer wg.Done()
		tracer.NewRootSpan("pylons.request", "pylons", "/").Finish()
	}()
	go func() {
		defer wg.Done()
		tracer.NewRootSpan("pylons.request", "pylons", "/home").Finish()
	}()
	go func() {
		defer wg.Done()
		tracer.NewRootSpan("pylons.request", "pylons", "/trace").Finish()
	}()

	wg.Wait()
	tracer.Flush()
	traces := transport.Traces()
	assert.Len(traces, 3)
	assert.Len(traces[0], 1)
	assert.Len(traces[1], 1)
	assert.Len(traces[2], 1)
}

func TestTracerConcurrentMultipleSpans(t *testing.T) {
	assert := assert.New(t)
	tracer, transport := getTestTracer()

	// Wait for two different goroutines that should create
	// two traces with two children each
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		parent := tracer.NewRootSpan("pylons.request", "pylons", "/")
		child := tracer.NewChildSpan("redis.command", parent)
		child.Finish()
		parent.Finish()
	}()
	go func() {
		defer wg.Done()
		parent := tracer.NewRootSpan("pylons.request", "pylons", "/")
		child := tracer.NewChildSpan("redis.command", parent)
		child.Finish()
		parent.Finish()
	}()

	wg.Wait()
	tracer.Flush()
	traces := transport.Traces()
	assert.Len(traces, 2)
	assert.Len(traces[0], 2)
	assert.Len(traces[1], 2)
}

// BenchmarkConcurrentTracing tests the performance of spawning a lot of
// goroutines where each one creates a trace with a parent and a child.
func BenchmarkConcurrentTracing(b *testing.B) {
	tracer, _ := getTestTracer()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		go func() {
			parent := tracer.NewRootSpan("pylons.request", "pylons", "/")
			defer parent.Finish()

			for i := 0; i < 10; i++ {
				tracer.NewChildSpan("redis.command", parent).Finish()
			}
		}()
	}
}

// BenchmarkTracerAddSpans tests the performance of creating and finishing a root
// span. It should include the encoding overhead.
func BenchmarkTracerAddSpans(b *testing.B) {
	tracer, _ := getTestTracer()

	for n := 0; n < b.N; n++ {
		span := tracer.NewRootSpan("pylons.request", "pylons", "/")
		span.Finish()
	}
}

// getTestTracer returns a Tracer with a DummyTransport
func getTestTracer() (*Tracer, *dummyTransport) {
	pool, _ := newEncoderPool(MSGPACK_ENCODER, encoderPoolSize)
	transport := &dummyTransport{pool: pool}
	tracer := NewTracerTransport(transport)
	return tracer, transport
}

// Mock Transport with a real Encoder
type dummyTransport struct {
	pool   *encoderPool
	traces [][]*Span
}

func (t *dummyTransport) Send(traces [][]*Span) (*http.Response, error) {
	t.traces = append(t.traces, traces...)
	encoder := t.pool.Borrow()
	defer t.pool.Return(encoder)
	return nil, encoder.Encode(traces)
}

func (t *dummyTransport) Traces() [][]*Span {
	traces := t.traces
	t.traces = nil
	return traces
}

func (t *dummyTransport) SetHeader(key, value string) {}
