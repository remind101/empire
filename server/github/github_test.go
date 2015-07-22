package github

import (
	"testing"

	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire/pkg/image"
)

func TestImage(t *testing.T) {
	tests := []struct {
		t   string
		d   events.Deployment
		out image.Image
	}{
		{DefaultTemplate, func() events.Deployment {
			var d events.Deployment
			d.Repository.FullName = "remind101/acme-inc"
			d.Deployment.Sha = "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"
			return d
		}(), image.Image{Repository: "remind101/acme-inc", Tag: "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"}},
	}

	for _, tt := range tests {
		img, err := Image(tt.t, tt.d)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := img, tt.out; got != want {
			t.Fatalf("Image => %v; want %v", got, want)
		}
	}
}
