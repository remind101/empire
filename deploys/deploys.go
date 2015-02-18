package deploys

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/manager"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

// ID represents the unique identifier for a Deploy.
type ID string

// Deploy represents a deployment to the platform.
type Deploy struct {
	ID      ID                `json:"id"`
	Status  string            `json:"status"`
	Release *releases.Release `json:"release"`
}

type Service struct {
	AppsService       *apps.Service
	ConfigsService    *configs.Service
	SlugsService      *slugs.Service
	ReleasesService   *releases.Service
	FormationsService *formations.Service
	ManagerService    *manager.Service
}

// Deploy deploys an Image to the platform.
func (s *Service) Deploy(image *slugs.Image) (*Deploy, error) {
	app, err := s.AppsService.FindOrCreateByRepo(image.Repo)
	if err != nil {
		return nil, err
	}

	// Grab the latest config.
	config, err := s.ConfigsService.Head(app)

	// Create a new slug for the docker image.
	//
	// TODO This is actually going to be pretty slow, so
	// we'll need to do
	// some polling or events/webhooks here.
	slug, err := s.SlugsService.CreateByImage(image)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	release, err := s.ReleasesService.Create(app, config, slug)
	if err != nil {
		return nil, err
	}

	// Get the current formation for the app.
	formations, err := s.FormationsService.Get(app)
	if err != nil {
		return nil, err
	}

	release.Formations = formations

	// Schedule the new release onto the cluster.
	if err := s.ManagerService.ScheduleRelease(release); err != nil {
		return nil, err
	}

	// We're deployed! ...
	// hopefully.
	return &Deploy{
		ID:      "1",
		Release: release,
	}, nil
}
