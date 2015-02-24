package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/processes"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

// ReleaseesService represents a service for interacting with Releases.
type ReleasesService interface {
	// Create creates a new release.
	Create(*apps.App, *configs.Config, *slugs.Slug) (*releases.Release, error)
}

// releasesService is a base implementation of the ReleasesService interface.
type releasesService struct {
	releases.Repository
	FormationsService FormationsService
}

// NewReleasesService returns a new ReleasesService instance.
func NewReleasesService(options Options, f FormationsService) (ReleasesService, error) {
	return &releasesService{
		Repository:        releases.NewRepository(),
		FormationsService: f,
	}, nil
}

// Create creates the release, then sets the current process formation on the release.
func (s *releasesService) Create(app *apps.App, config *configs.Config, slug *slugs.Slug) (*releases.Release, error) {
	// Create a new formation for this release.
	formation, err := s.createFormation(app, slug)
	if err != nil {
		return nil, err
	}

	r := &releases.Release{
		App:       app,
		Config:    config,
		Slug:      slug,
		Formation: formation,
	}

	return s.Repository.Create(r)
}

func (s *releasesService) createFormation(app *apps.App, slug *slugs.Slug) (*formations.Formation, error) {
	// Get the old release, so we can copy the Formation.
	old, err := s.Repository.Head(app.Name)
	if err != nil {
		return nil, err
	}

	var p processes.ProcessMap
	if old != nil {
		p = old.Formation.Processes
	}

	formation := &formations.Formation{
		Processes: processes.NewProcessMap(p, slug.ProcessTypes),
	}

	return s.FormationsService.Create(formation)
}
