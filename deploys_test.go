package empire

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

func TestDeploysServiceDeploy(t *testing.T) {
	var released bool

	a := &mockAppsService{}
	c := &mockConfigsService{}
	s := &mockSlugsService{}
	r := &mockReleasesService{
		CreateFunc: func(app *apps.App, config *configs.Config, slug *slugs.Slug) (*releases.Release, error) {
			released = true
			return nil, nil
		},
	}

	d := &deploysService{
		AppsService:     a,
		ConfigsService:  c,
		SlugsService:    s,
		ReleasesService: r,
	}

	image := &images.Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	}

	if _, err := d.Deploy(image); err != nil {
		t.Fatal(err)
	}

	if got, want := released, true; got != want {
		t.Fatal("Expected a release to be created")
	}
}
