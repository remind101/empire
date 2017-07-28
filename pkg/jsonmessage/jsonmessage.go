// Package jsonmessage is a stripped down version of
// github.com/docker/docker/pkg/jsonmessage without any transitive dependencies.
package jsonmessage

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

func (e *JSONError) Error() string {
	return e.Message
}
