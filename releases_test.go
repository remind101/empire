package empire

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

func TestReleasesServiceCreate(t *testing.T) {
	f, err := NewFormationsService(DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}
	s, err := NewReleasesService(DefaultOptions, f)
	if err != nil {
		t.Fatal(err)
	}

	app := &apps.App{Name: "api"}
	config := &configs.Config{}
	slug := &slugs.Slug{
		ProcessTypes: slugs.ProcessMap{
			"web": "./bin/web",
		},
	}

	r, err := s.Create(app, config, slug)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Formation) != 1 {
		t.Fatal("Expected an initial process formation")
	}
}

type mockReleasesService struct {
	ReleasesService // Just to satisfy the interface.

	CreateFunc func(*apps.App, *configs.Config, *slugs.Slug) (*releases.Release, error)
}

func (s *mockReleasesService) Create(app *apps.App, config *configs.Config, slug *slugs.Slug) (*releases.Release, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(app, config, slug)
	}

	return nil, nil
}
