package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
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
	r, err := s.Repository.Create(app, config, slug)
	if err != nil {
		return r, err
	}

	// Get the currently configured process formation, or create a new one
	// based on the slugs process types if the app doesn't already have a
	// process formation.
	fmtns, err := s.FormationsService.GetOrCreate(app, slug)
	if err != nil {
		return r, err
	}

	for _, f := range fmtns {
		cmd, found := slug.ProcessTypes[f.ProcessType]
		if !found {
			// TODO Update the formation?
			continue
		}

		r.Formation = append(r.Formation, &formations.CommandFormation{
			Formation: f,
			Command:   cmd,
		})
	}

	return r, nil
}
