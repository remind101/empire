package dockerutil

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/remind101/empire/pkg/image"
	"github.com/stretchr/testify/assert"
)

func TestPullImageOptions(t *testing.T) {
	img, _ := image.Decode("remind101/acme-inc:latest")
	options, err := PullImageOptions(img)
	assert.NoError(t, err)
	assert.Equal(t, "remind101/acme-inc", options.Repository)
	assert.Equal(t, "", options.Registry)
	assert.Equal(t, "latest", options.Tag)

	img, _ = image.Decode("busybox:latest")
	options, err = PullImageOptions(img)
	assert.NoError(t, err)
	assert.Equal(t, "busybox", options.Repository)
	assert.Equal(t, "", options.Registry)
	assert.Equal(t, "latest", options.Tag)

	img, _ = image.Decode("quay.io/remind101/acme-inc:latest")
	options, err = PullImageOptions(img)
	assert.NoError(t, err)
	assert.Equal(t, "remind101/acme-inc", options.Repository)
	assert.Equal(t, "quay.io", options.Registry)
	assert.Equal(t, "latest", options.Tag)

	img, _ = image.Decode("busybox@sha256:7d3ce4e482101f0c484602dd6687c826bb8bef6295739088c58e84245845912e")
	options, err = PullImageOptions(img)
	assert.NoError(t, err)
	assert.Equal(t, "busybox", options.Repository)
	assert.Equal(t, "", options.Registry)
	assert.Equal(t, "sha256:7d3ce4e482101f0c484602dd6687c826bb8bef6295739088c58e84245845912e", options.Tag)
}

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

	err := FakePull(image.Image{Repository: "remind101/acme-inc", Tag: "latest"}, w)
	assert.NoError(t, err)
	assert.Equal(t, "Pulling repository remind101/acme-inc\n345c7524bc96: Pulling image (latest) from remind101/acme-inc\n345c7524bc96: Pulling image (latest) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/\n345c7524bc96: Pulling dependent layers\na1dd7097a8e8: Download complete\nStatus: Image is up to date for remind101/acme-inc:latest\n", buf.String())
}
