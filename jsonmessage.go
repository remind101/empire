package empire

import (
	"encoding/json"
	"io"

	"github.com/docker/docker/pkg/jsonmessage"
)

// JSONStream represents an stream of jsonmessage.JSONMessage objects.
// This is used for streaming deployment logs.
type JSONStream struct {
	// Implements the io.Writer interface for anything outside of this
	// pacakge to encode jsonmessage objects. (e.g. docker client).
	io.Writer

	enc *json.Encoder
}

// NewJSONStream returns a new JSONStream instance that writes jsonmessage
// objects to w.
func NewJSONStream(w io.Writer) *JSONStream {
	return &JSONStream{
		Writer: w,
		enc:    json.NewEncoder(w),
	}
}

// Encode encodes the JSONMessage into the stream.
func (w *JSONStream) Encode(m jsonmessage.JSONMessage) error {
	return w.enc.Encode(m)
}

// Writes a JSONMessage encoded error into the stream.
func (w *JSONStream) Error(err error) error {
	return w.Encode(jsonmessage.JSONMessage{
		ErrorMessage: err.Error(),
		Error: &jsonmessage.JSONError{
			Message: err.Error(),
		},
	})
}

// Writes a JSONMessage status into the stream.
func (w *JSONStream) Status(message string) error {
	return w.Encode(jsonmessage.JSONMessage{Status: message})
}
