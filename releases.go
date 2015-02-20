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
	releases.Repository
}

// releasesService is a base implementation of the ReleasesService interface.
type releasesService struct {
	releases.Repository
	FormationsService *formations.Service
}

// NewReleasesService returns a new ReleasesService instance.
func NewReleasesService(r releases.Repository, f *formations.Service) ReleasesService {
	if r == nil {
		r = releases.NewRepository()
	}

	return &releasesService{
		Repository:        r,
		FormationsService: f,
	}
}

// Create creates the release, then sets the current process formation on the release.
func (s *releasesService) Create(app *apps.App, config *configs.Config, slug *slugs.Slug) (*releases.Release, error) {
	r, err := s.Repository.Create(app, config, slug)
	if err != nil {
		return r, err
	}

	// Get the currently configured process formation.
	fmtns, err := s.FormationsService.Get(app)
	if err != nil {
		return r, err
	}

	if fmtns == nil {
		if _, ok := slug.ProcessTypes["web"]; ok {
			fmtns = formations.Formations{
				"web": formations.NewFormation("web"),
			}

			if err := s.FormationsService.Set(app, fmtns); err != nil {
				return nil, err
			}
		}
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
