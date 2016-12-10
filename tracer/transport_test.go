package tracer

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// getTestSpan returns a Span with different fields set
func getTestSpan() *Span {
	return &Span{
		TraceID:  42,
		SpanID:   52,
		ParentID: 42,
		Type:     "web",
		Service:  "high.throughput",
		Name:     "sending.events",
		Resource: "SEND /data",
		Start:    1481215590883401105,
		Duration: 1000000000,
		Meta:     map[string]string{"http.host": "192.168.0.1"},
		Metrics:  map[string]float64{"http.monitor": 41.99},
	}
}

// getTestTrace returns a list of traces that is composed by ``traceN`` number
// of traces, each one composed by ``size`` number of spans.
func getTestTrace(traceN, size int) [][]*Span {
	var traces [][]*Span

	for i := 0; i < traceN; i++ {
		trace := []*Span{}
		for j := 0; j < size; j++ {
			trace = append(trace, getTestSpan())
		}
		traces = append(traces, trace)
	}
	return traces
}

func TestTransportHeaders(t *testing.T) {
	assert := assert.New(t)
	transport := NewHTTPTransport(defaultDeliveryURL)

	// msgpack is the default Header
	contentType := transport.headers["Content-Type"]
	assert.Equal("application/msgpack", contentType)
}

func TestTransportEncoderPool(t *testing.T) {
	assert := assert.New(t)
	transport := NewHTTPTransport(defaultDeliveryURL)

	// MsgpackEncoder is the default encoder of the pool
	encoder := transport.pool.Borrow()
	assert.Equal("application/msgpack", encoder.ContentType())
}

func TestTransportSwitchEncoder(t *testing.T) {
	assert := assert.New(t)
	transport := NewHTTPTransport(defaultDeliveryURL)
	transport.changeEncoder(JSON_ENCODER)

	// MsgpackEncoder is the default encoder of the pool
	encoder := transport.pool.Borrow()
	contentType := transport.headers["Content-Type"]
	assert.Equal("application/json", encoder.ContentType())
	assert.Equal("application/json", contentType)
}
