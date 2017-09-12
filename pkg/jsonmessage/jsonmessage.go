// Package jsonmessage is a stripped down version of
// github.com/docker/docker/pkg/jsonmessage without any transitive dependencies.
package jsonmessage

import (
	"encoding/json"
	"io"
)

// Stream provides a simple wrapper around an io.Writer to write jsonmessage's
// to it.
type Stream struct {
	io.Writer
	enc *json.Encoder
}

// NewStream returns a new Stream backed by w.
func NewStream(w io.Writer) *Stream {
	return &Stream{
		Writer: w,
		enc:    json.NewEncoder(w),
	}
}

// Encode encodes m into the stream and implements the jsonmessage.Writer
// interface.
func (w *Stream) Encode(m JSONMessage) error {
	return w.enc.Encode(m)
}

type JSONMessage struct {
	Status       string     `json:"status,omitempty"`
	Error        *JSONError `json:"errorDetail,omitempty"`
	ErrorMessage string     `json:"error,omitempty"` //deprecated
}

// JSONError wraps a concrete Code and Message, `Code` is
// is a integer error code, `Message` is the error message.
type JSONError struct {
	Code    int    `json:"code,omitempty"`
	Message string `json:"message,omitempty"`
}

func NewError(err error) JSONMessage {
	return JSONMessage{
		ErrorMessage: err.Error(),
		Error: &JSONError{
			Message: err.Error(),
		},
	}
}

func (e *JSONError) Error() string {
	return e.Message
}
