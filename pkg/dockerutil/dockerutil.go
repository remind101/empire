package dockerutil

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
	docker "github.com/fsouza/go-dockerclient"
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

// PullImageOptions generates an appropriate docker.PullImageOptions to pull the
// target image based on the idiosyncrasies of docker.
func PullImageOptions(img image.Image) (docker.PullImageOptions, error) {
	var options docker.PullImageOptions

	// From the Docker API docs:
	//
	//	Tag or digest. If empty when pulling an image, this
	//	causes all tags for the given image to be pulled.
	//
	// So, we prefer the digest if it's provided.
	tag := img.Digest
	if tag == "" {
		tag = img.Tag
	}

	// If there's no tag or digest, error out. Providing an empty
	// tag to DockerPull will pull all images, which we don't want.
	if tag == "" {
		return options, fmt.Errorf("no tag or digest provided")
	}

	options.Tag = tag
	options.Repository = img.Repository

	// Only required for Docker Engine 1.9 or 1.10 w/ Remote API < 1.21
	// and Docker Engine < 1.9
	// This parameter was removed in Docker Engine 1.11
	//
	// See https://goo.gl/9y9Bpx
	options.Registry = img.Registry

	return options, nil
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
