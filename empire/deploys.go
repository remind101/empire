package empire

import "fmt"

// DeployID represents the unique identifier for a Deploy.
type DeployID string

// Deploy represents a deployment to the platform.
type Deploy struct {
	ID      DeployID `json:"id"`
	Status  string   `json:"status"`
	Release *Release `json:"release"`
}

// DeploysService is a base implementation of the DeploysService
type DeploysService struct {
	AppsService
	ConfigsService
	SlugsService
	ReleasesService
}

// Deploy deploys an Image to the cluster.
func (s *DeploysService) DeployImage(image Image) (*Deploy, error) {
	app, err := s.AppsService.AppsFindOrCreateByRepo(image.Repo)
	if err != nil {
		return nil, err
	}

	// Grab the latest config.
	config, err := s.ConfigsService.ConfigsCurrent(app)
	if err != nil {
		return nil, err
	}

	// Create a new slug for the docker image.
	//
	// TODO This is actually going to be pretty slow, so
	// we'll need to do
	// some polling or events/webhooks here.
	slug, err := s.SlugsService.SlugsCreateByImage(image)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	desc := fmt.Sprintf("Deploy %s", image.String())
	release, err := s.ReleasesService.ReleasesCreate(app, config, slug, desc)
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
