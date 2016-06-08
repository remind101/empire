package dockerutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/stretchr/testify/assert"
)

func TestDecodeJSONMessageStream(t *testing.T) {
	buf := new(bytes.Buffer)
	w := DecodeJSONMessageStream(buf)

	err := json.NewEncoder(w).Encode(&jsonmessage.JSONMessage{
		Status: "Doing stuff",
	})
	assert.NoError(t, err)
	assert.Equal(t, "Doing stuff\n", buf.String())
}

func TestDecodeJSONMessageStream_JSONMessageError(t *testing.T) {
	buf := new(bytes.Buffer)
	w := DecodeJSONMessageStream(buf)

	err := json.NewEncoder(w).Encode(&jsonmessage.JSONMessage{
		Error: &jsonmessage.JSONError{
			Message: "error message",
		},
	})
	assert.NoError(t, err)
	assert.EqualError(t, w.Err(), "error message")
}

func TestDecodeJSONMessageStream_DockerPull(t *testing.T) {
	buf := new(bytes.Buffer)
	w := DecodeJSONMessageStream(buf)

	err := FakePull("remind101/acme-inc:latest", w)
	assert.NoError(t, err)
	assert.Equal(t, "Pulling repository remind101/acme-inc\n345c7524bc96: Pulling image (latest) from remind101/acme-inc\n345c7524bc96: Pulling image (latest) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/\n345c7524bc96: Pulling dependent layers\na1dd7097a8e8: Download complete\nStatus: Image is up to date for remind101/acme-inc:latest\n", buf.String())
}
