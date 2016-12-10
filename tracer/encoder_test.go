package tracer

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/ugorji/go/codec"
)

func TestEncoderContentType(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		encoder     Encoder
		contentType string
	}{
		{newJSONEncoder(), "application/json"},
		{newMsgpackEncoder(), "application/msgpack"},
	}

	for _, tc := range testCases {
		assert.Equal(tc.contentType, tc.encoder.ContentType())
	}
}

func TestJSONEncoding(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		traces int
		size   int
	}{
		{1, 1},
		{3, 1},
		{1, 3},
		{3, 3},
	}

	for _, tc := range testCases {
		payload := getTestTrace(tc.traces, tc.size)
		encoder := newJSONEncoder()
		err := encoder.Encode(payload)
		assert.Nil(err)

		// decode to check the right encoding
		var traces [][]*Span
		dec := json.NewDecoder(encoder.buffer)
		err = dec.Decode(&traces)
		assert.Nil(err)
		assert.Len(traces, tc.traces)

		for _, trace := range traces {
			assert.Len(trace, tc.size)
			span := trace[0]
			assert.Equal(uint64(42), span.TraceID)
			assert.Equal(uint64(52), span.SpanID)
			assert.Equal(uint64(42), span.ParentID)
			assert.Equal("web", span.Type)
			assert.Equal("high.throughput", span.Service)
			assert.Equal("sending.events", span.Name)
			assert.Equal("SEND /data", span.Resource)
			assert.Equal(int64(1481215590883401105), span.Start)
			assert.Equal(int64(1000000000), span.Duration)
			assert.Equal("192.168.0.1", span.Meta["http.host"])
			assert.Equal(float64(41.99), span.Metrics["http.monitor"])
		}
	}
}

func TestMsgpackEncoding(t *testing.T) {
	assert := assert.New(t)

	testCases := []struct {
		traces int
		size   int
	}{
		{1, 1},
		{3, 1},
		{1, 3},
		{3, 3},
	}

	for _, tc := range testCases {
		payload := getTestTrace(tc.traces, tc.size)
		encoder := newMsgpackEncoder()
		err := encoder.Encode(payload)
		assert.Nil(err)

		// decode to check the right encoding
		var traces [][]*Span
		var mh codec.MsgpackHandle
		dec := codec.NewDecoder(encoder.buffer, &mh)
		err = dec.Decode(&traces)
		assert.Nil(err)
		assert.Len(traces, tc.traces)

		for _, trace := range traces {
			assert.Len(trace, tc.size)
			span := trace[0]
			assert.Equal(uint64(42), span.TraceID)
			assert.Equal(uint64(52), span.SpanID)
			assert.Equal(uint64(42), span.ParentID)
			assert.Equal("web", span.Type)
			assert.Equal("high.throughput", span.Service)
			assert.Equal("sending.events", span.Name)
			assert.Equal("SEND /data", span.Resource)
			assert.Equal(int64(1481215590883401105), span.Start)
			assert.Equal(int64(1000000000), span.Duration)
			assert.Equal("192.168.0.1", span.Meta["http.host"])
			assert.Equal(float64(41.99), span.Metrics["http.monitor"])
		}
	}
}

func TestPoolBorrowCreate(t *testing.T) {
	assert := assert.New(t)

	// borrow an encoder from the pool
	pool, _ := newEncoderPool(MSGPACK_ENCODER, 1)
	encoder := pool.Borrow()
	assert.NotNil(encoder)
}

func TestPoolReuseEncoder(t *testing.T) {
	assert := assert.New(t)

	// borrow, return and borrow again an encoder from the pool
	pool, _ := newEncoderPool(MSGPACK_ENCODER, 1)
	encoder := pool.Borrow()
	pool.Return(encoder)
	anotherEncoder := pool.Borrow()
	assert.Equal(anotherEncoder, encoder)
}

func TestPoolSize(t *testing.T) {
	pool, _ := newEncoderPool(MSGPACK_ENCODER, 1)
	encoder := newMsgpackEncoder()
	anotherEncoder := newMsgpackEncoder()

	// put two encoders in the pool with a maximum size of 1
	// doesn't hang the caller
	pool.Return(encoder)
	pool.Return(anotherEncoder)
}

func TestPoolReturn(t *testing.T) {
	assert := assert.New(t)

	// an encoder can return in the pool
	pool, _ := newEncoderPool(MSGPACK_ENCODER, 5)
	encoder := newMsgpackEncoder()
	pool.pool <- encoder
	pool.Return(encoder)

	// the encoder is the one we get before
	returnedEncoder := <-pool.pool
	assert.Equal(returnedEncoder, encoder)
}
