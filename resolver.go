package empire

import (
	"encoding/json"
	"io"

	"golang.org/x/net/context"

	"github.com/remind101/empire/pkg/image"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
)

type Resolver interface {
	Resolve(context.Context, image.Image, chan Event) (image.Image, error)
}

// fakeResolver is a fake resolver that will just return the provided image.
type fakeResolver struct{}

func (r *fakeResolver) Resolve(_ context.Context, img image.Image, out chan Event) (image.Image, error) {
	for _, e := range FakeDockerPull(img) {
		ee := e
		out <- &ee
	}
	return img, nil
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

func (r *dockerResolver) Resolve(ctx context.Context, img image.Image, out chan Event) (image.Image, error) {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		errCh <- r.pullImage(ctx, img, pw)
	}()

	dec := json.NewDecoder(pr)
	for {
		var e DockerEvent
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return img, err
		}
		out <- &e
	}

	// Wait for pullImage to finish
	if err := <-errCh; err != nil {
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
