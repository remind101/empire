package empire

import (
	"encoding/json"
	"fmt"
	"io"

	"golang.org/x/net/context"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/image"
)

type Resolver interface {
	Resolve(context.Context, image.Image, io.Writer) (image.Image, error)
}

// fakeResolver is a fake resolver that will just return the provided image.
type fakeResolver struct{}

func (r *fakeResolver) Resolve(_ context.Context, img image.Image, out io.Writer) (image.Image, error) {
	err := FakeDockerPull(img, out)
	return img, err
}

// dockerResolver is a resolver that pulls the docker image, then inspects it to
// get the canonical image id.
type dockerResolver struct {
	client *dockerutil.Client
}

func newDockerResolver(c *dockerutil.Client) Resolver {
	return &dockerResolver{
		client: c,
	}
}

func (r *dockerResolver) Resolve(ctx context.Context, img image.Image, out io.Writer) (image.Image, error) {
	if err := r.pullImage(ctx, img, out); err != nil {
		return img, err
	}

	i, err := r.client.InspectImage(img.String())
	if err != nil {
		return img, err
	}

	return image.Image{
		Repository: img.Repository,
		Tag:        i.ID,
	}, nil
}

// pullImage can pull a docker image from a repo, by its imageID.
//
// Because docker does not support pulling an image by ID, we're assuming that
// the docker image has been tagged with its own ID beforehand.
func (r *dockerResolver) pullImage(ctx context.Context, img image.Image, output io.Writer) error {
	return r.client.PullImage(ctx, docker.PullImageOptions{
		Registry:      img.Registry,
		Repository:    img.Repository,
		Tag:           img.Tag,
		OutputStream:  output,
		RawJSONStream: true,
	})
}

// FakeDockerPull returns a slice of events that look like a docker pull.
func FakeDockerPull(img image.Image, w io.Writer) error {
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
