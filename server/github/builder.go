package github

import (
	"bytes"
	"text/template"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

// ImageBuilder is an interface that represents something that can build and
// return a Docker image from a GitHub commit.
type ImageBuilder interface {
	BuildImage(ctx context.Context, event events.Deployment) (image.Image, error)
}

type ImageBuilderFunc func(context.Context, events.Deployment) (image.Image, error)

func (fn ImageBuilderFunc) BuildImage(ctx context.Context, event events.Deployment) (image.Image, error) {
	return fn(ctx, event)
}

// ImageFromTemplate returns an ImageBuilder that will execute a template to
// determine what docker image should be deployed. Note that this doesn't not
// actually perform any "build".
func ImageFromTemplate(t *template.Template) ImageBuilder {
	return ImageBuilderFunc(func(ctx context.Context, event events.Deployment) (image.Image, error) {
		buf := new(bytes.Buffer)
		if err := t.Execute(buf, event); err != nil {
			return image.Image{}, err
		}

		return image.Decode(buf.String())
	})
}
