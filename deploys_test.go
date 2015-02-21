package empire

import (
	"testing"

	"github.com/remind101/empire/images"
	"github.com/remind101/empire/releases"
)

func TestDeploysServiceDeploy(t *testing.T) {
	var scheduled bool

	a := &mockAppsService{}
	c := &mockConfigsService{}
	s := &mockSlugsService{}
	r := &mockReleasesService{}
	m := &mockManager{
		ScheduleReleaseFunc: func(release *releases.Release) error {
			scheduled = true
			return nil
		},
	}

	d := &deploysService{
		AppsService:     a,
		ConfigsService:  c,
		SlugsService:    s,
		ReleasesService: r,
		Manager:         m,
	}

	image := &images.Image{
		Repo: "remind101/r101-api",
		ID:   "1234",
	}

	if _, err := d.Deploy(image); err != nil {
		t.Fatal(err)
	}

	if got, want := scheduled, true; got != want {
		t.Fatal("Expected a release to be scheduled")
	}
}
