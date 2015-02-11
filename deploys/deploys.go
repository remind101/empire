package deploys

import (
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

// Deploy represents a deployment to the platform.
type Deploy struct {
	ID      string
	Status  string
	Release *releases.Release
}

type DeploysService struct {
	ConfigsService  configs.ConfigService
	SlugsService    slugs.SlugsService
	ReleasesService releases.ReleasesService
}

// Deploy deploys an Image to the platform.
func (s *DeploysService) Deploy(image *slugs.Image) (*Deploy, error) {
	// Grab the latest config.
	config, err := s.ConfigsService.Head(image.Repo)
	if err != nil {
		return nil, err
	}

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
	release, err := s.ReleasesService.Create(config, slug)
	if err != nil {
		return nil, err
	}

	// We're deployed! ...
	// hopefully.
	return &Deploy{
		ID:      "1",
		Release: release,
	}, nil
}
