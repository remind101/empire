package dockerutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	"github.com/remind101/empire/pkg/image"
)

// FakePull writes a jsonmessage stream to w that looks like a Docker pull.
func FakePull(img image.Image, w io.Writer) error {
	messages := []jsonmessage.JSONMessage{
		{Status: fmt.Sprintf("Pulling repository %s", img.Repository)},
		{Status: fmt.Sprintf("Pulling image (%s) from %s", img.Tag, img.Repository), Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: fmt.Sprintf("Pulling image (%s) from %s, endpoint: https://registry-1.docker.io/v1/", img.Tag, img.Repository), Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: "Pulling dependent layers", Progress: &jsonmessage.JSONProgress{}, ID: "345c7524bc96"},
		{Status: "Download complete", Progress: &jsonmessage.JSONProgress{}, ID: "a1dd7097a8e8"},
		{Status: fmt.Sprintf("Status: Image is up to date for %s", img)},
	}

	enc := json.NewEncoder(w)

	for _, m := range messages {
		if err := enc.Encode(&m); err != nil {
			return err
		}
	}

	return nil
}

// DecodeJSONMessageStream wraps an io.Writer to decode a jsonmessage stream into
// plain text. Bytes written to w represent the decoded plain text stream.
func DecodeJSONMessageStream(w io.Writer) *DecodedJSONMessageWriter {
	outFd, _ := term.GetFdInfo(w)
	return &DecodedJSONMessageWriter{
		w:  w,
		fd: outFd,
	}
}

// DecodedJSONMessageWriter is an io.Writer that decodes a jsonmessage stream.
type DecodedJSONMessageWriter struct {
	// The wrapped io.Writer. The plain text stream will be written here.
	w  io.Writer
	fd uintptr

	// err holds the error returned after the jsonmessage stream is
	// completely read.
	err error
}

// Write decodes the jsonmessage stream in the bytes, and writes the decoded
// plain text to the underlying io.Writer.
func (w *DecodedJSONMessageWriter) Write(b []byte) (int, error) {
	err := jsonmessage.DisplayJSONMessagesStream(bytes.NewReader(b), w.w, w.fd, false, nil)
	if err != nil {
		if err, ok := err.(*jsonmessage.JSONError); ok {
			w.err = err
			return len(b), nil
		}
	}
	return len(b), err
}

// Err returns the jsonmessage.Error that occurred, if any.
func (w *DecodedJSONMessageWriter) Err() error {
	return w.err
}
