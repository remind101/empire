package github

import (
	"bytes"
	"io"
	"text/template"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/conveyor/client/conveyor"
	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

// ImageBuilder is an interface that represents something that can build and
// return a Docker image from a GitHub commit.
type ImageBuilder interface {
	BuildImage(ctx context.Context, w io.Writer, event events.Deployment) (image.Image, error)
}

type ImageBuilderFunc func(context.Context, io.Writer, events.Deployment) (image.Image, error)

func (fn ImageBuilderFunc) BuildImage(ctx context.Context, w io.Writer, event events.Deployment) (image.Image, error) {
	return fn(ctx, w, event)
}

// ImageFromTemplate returns an ImageBuilder that will execute a template to
// determine what docker image should be deployed. Note that this doesn't not
// actually perform any "build".
func ImageFromTemplate(t *template.Template) ImageBuilder {
	return ImageBuilderFunc(func(ctx context.Context, _ io.Writer, event events.Deployment) (image.Image, error) {
		buf := new(bytes.Buffer)
		if err := t.Execute(buf, event); err != nil {
			return image.Image{}, err
		}

		return image.Decode(buf.String())
	})
}

// conveyorClient mocks the interface to the Conveyor API.
type conveyorClient interface {
	Build(io.Writer, conveyor.BuildCreateOpts) (*conveyor.Artifact, error)
}

// ConveyorImageBuilder provides an ImageBuilder implementation that
// integrations with the Conveyor (https://github.com/remind101/conveyor) Docker
// build system. If enabled, Empire will check if an Artifact in Conveyor exists
// for the git commit, and will trigger Conveyor to build it if it doesn't
// exist.
type ConveyorImageBuilder struct {
	client conveyorClient
}

// NewConveyorImageBuilder returns a new ConveyorImageBuilder implementation
// that uses the given client.
func NewConveyorImageBuilder(c *conveyor.Service) *ConveyorImageBuilder {
	return &ConveyorImageBuilder{
		client: c,
	}
}

func (c *ConveyorImageBuilder) BuildImage(ctx context.Context, w io.Writer, event events.Deployment) (image.Image, error) {
	a, err := c.client.Build(w, conveyor.BuildCreateOpts{
		Repository: event.Repository.FullName,
		Sha:        &event.Deployment.Sha,
	})
	if err != nil {
		return image.Image{}, err
	}

	return image.Decode(a.Image)
}
