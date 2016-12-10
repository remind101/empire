package tracer

import (
	"bytes"
	"encoding/json"

	"github.com/ugorji/go/codec"
)

// Encoder is a generic interface that expects an Encode() method
// for the encoding process, and a Read() method that will be used
// by the http handler
type Encoder interface {
	Encode(traces [][]*Span) error
	Read(p []byte) (int, error)
	ContentType() string
}

var mh codec.MsgpackHandle

// msgpackEncoder encodes a list of traces in Msgpack format
type msgpackEncoder struct {
	buffer      *bytes.Buffer
	encoder     *codec.Encoder
	contentType string
}

func newMsgpackEncoder() *msgpackEncoder {
	buffer := &bytes.Buffer{}
	encoder := codec.NewEncoder(buffer, &mh)

	return &msgpackEncoder{
		buffer:      buffer,
		encoder:     encoder,
		contentType: "application/msgpack",
	}
}

// Encode serializes the given traces list into the internal
// buffer, returning the error if any
func (e *msgpackEncoder) Encode(traces [][]*Span) error {
	e.buffer.Reset()
	return e.encoder.Encode(traces)
}

// Read values from the internal buffer
func (e *msgpackEncoder) Read(p []byte) (int, error) {
	return e.buffer.Read(p)
}

// ContentType return the msgpackEncoder content-type
func (e *msgpackEncoder) ContentType() string {
	return e.contentType
}

// jsonEncoder encodes a list of traces in JSON format
type jsonEncoder struct {
	buffer      *bytes.Buffer
	encoder     *json.Encoder
	contentType string
}

// newJSONEncoder returns a new encoder for the JSON format.
func newJSONEncoder() *jsonEncoder {
	buffer := &bytes.Buffer{}
	encoder := json.NewEncoder(buffer)

	return &jsonEncoder{
		buffer:      buffer,
		encoder:     encoder,
		contentType: "application/json",
	}
}

// Encode serializes the given traces list into the internal
// buffer, returning the error if any
func (e *jsonEncoder) Encode(traces [][]*Span) error {
	e.buffer.Reset()
	return e.encoder.Encode(traces)
}

// Read values from the internal buffer
func (e *jsonEncoder) Read(p []byte) (int, error) {
	return e.buffer.Read(p)
}

// ContentType return the jsonEncoder content-type
func (e *jsonEncoder) ContentType() string {
	return e.contentType
}

const (
	JSON_ENCODER = iota
	MSGPACK_ENCODER
)

// EncoderPool is a pool meant to share the buffers required to encode traces.
// It naively tries to cap the number of active encoders, but doesn't enforce
// the limit. To use a pool, you should Borrow() for an encoder and then
// Return() that encoder to the pool. Encoders in that pool should honor
// the Encoder interface.
type encoderPool struct {
	encoderType int
	pool        chan Encoder
}

func newEncoderPool(encoderType, size int) (*encoderPool, string) {
	pool := &encoderPool{
		encoderType: encoderType,
		pool:        make(chan Encoder, size),
	}

	// Borrow an encoder to retrieve the default ContentType
	encoder := pool.Borrow()
	pool.Return(encoder)

	contentType := encoder.ContentType()
	return pool, contentType
}

func (p *encoderPool) Borrow() Encoder {
	var encoder Encoder

	select {
	case encoder = <-p.pool:
	default:
		switch p.encoderType {
		case JSON_ENCODER:
			encoder = newJSONEncoder()
		case MSGPACK_ENCODER:
			encoder = newMsgpackEncoder()
		}
	}
	return encoder
}

func (p *encoderPool) Return(e Encoder) {
	select {
	case p.pool <- e:
	default:
	}
}
