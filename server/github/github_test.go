package github

import (
	"context"
	"io/ioutil"
	"testing"
	"text/template"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire/pkg/image"
	"github.com/stretchr/testify/assert"
)

var defaultTemplate = template.Must(template.New("image").Parse(DefaultTemplate))

func TestDefaultTemplate(t *testing.T) {
	b := ImageFromTemplate(defaultTemplate)

	tests := []struct {
		d   events.Deployment
		out image.Image
	}{
		{func() events.Deployment {
			var d events.Deployment
			d.Repository.FullName = "remind101/acme-inc"
			d.Deployment.Sha = "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"
			return d
		}(), image.Image{Repository: "remind101/acme-inc", Tag: "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"}},
	}

	for _, tt := range tests {
		img, err := b.BuildImage(context.Background(), ioutil.Discard, tt.d)
		assert.NoError(t, err)
		assert.Equal(t, tt.out, img)
	}
}
