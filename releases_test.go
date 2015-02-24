package empire

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

func TestReleasesServiceCreate(t *testing.T) {
	var scheduled bool

	app := &apps.App{}
	config := &configs.Config{}
	slug := &slugs.Slug{}

	f := &mockFormationsRepository{}
	r := &mockReleasesRepository{}
	m := &mockManager{
		ScheduleReleaseFunc: func(release *releases.Release) error {
			scheduled = true
			return nil
		},
	}
	s := &releasesService{
		Repository:           r,
		FormationsRepository: f,
		Manager:              m,
	}

	if _, err := s.Create(app, config, slug); err != nil {
		t.Fatal(err)
	}

	if got, want := scheduled, true; got != want {
		t.Fatal("Expected a release to be created")
	}
}

type mockReleasesRepository struct {
	releases.Repository // Just to satisfy the interface.

	HeadFunc   func(apps.Name) (*releases.Release, error)
	CreateFunc func(*releases.Release) (*releases.Release, error)
}

func (s *mockReleasesRepository) Head(name apps.Name) (*releases.Release, error) {
	if s.HeadFunc != nil {
		return s.HeadFunc(name)
	}

	return nil, nil
}

func (s *mockReleasesRepository) Create(release *releases.Release) (*releases.Release, error) {
	if s.CreateFunc != nil {
		return s.CreateFunc(release)
	}

	return release, nil
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
