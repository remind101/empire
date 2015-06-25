package empire

import (
	"encoding/json"
	"io"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/empire/pkg/dockerutil"
)

type Resolver interface {
	Resolve(Image, chan Event) (Image, error)
}

// fakeResolver is a fake resolver that will just return the provided image.
type fakeResolver struct{}

func (r *fakeResolver) Resolve(image Image, out chan Event) (Image, error) {
	for _, e := range FakeDockerPull(image) {
		ee := e
		out <- &ee
	}
	return image, nil
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

func (r *dockerResolver) Resolve(image Image, out chan Event) (Image, error) {
	pr, pw := io.Pipe()
	errCh := make(chan error, 1)
	go func() {
		defer pw.Close()
		errCh <- r.pullImage(image, pw)
	}()

	dec := json.NewDecoder(pr)
	for {
		var e DockerEvent
		if err := dec.Decode(&e); err == io.EOF {
			break
		} else if err != nil {
			return image, err
		}
		out <- &e
	}

	// Wait for pullImage to finish
	if err := <-errCh; err != nil {
		return image, err
	}

	i, err := r.client.InspectImage(image.String())
	if err != nil {
		return image, err
	}

	return Image{
		Repo: image.Repo,
		ID:   i.ID,
	}, nil
}

// pullImage can pull a docker image from a repo, by its imageID.
//
// Because docker does not support pulling an image by ID, we're assuming that
// the docker image has been tagged with its own ID beforehand.
func (r *dockerResolver) pullImage(i Image, output io.Writer) error {
	return r.client.PullImage(docker.PullImageOptions{
		Repository:    string(i.Repo),
		Tag:           i.ID,
		OutputStream:  output,
		RawJSONStream: true,
	})
}
