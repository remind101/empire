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

// DeploysService is an interface that can be implemented to deploy images.
type DeploysService interface {
	// Deploy deploys a container image to the cluster.
	Deploy(Image) (*Deploy, error)
}

// deploysService is a base implementation of the DeploysService
type deploysService struct {
	AppsService
	ConfigsService
	SlugsService
	ReleasesService
}

// Deploy deploys an Image to the cluster.
func (s *deploysService) Deploy(image Image) (*Deploy, error) {
	app, err := s.AppsService.FindOrCreateByRepo(image.Repo)
	if err != nil {
		return nil, err
	}

	// Grab the latest config.
	config, err := s.ConfigsService.Head(app)
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
	desc := fmt.Sprintf("Deploy %s", image.String())
	release, err := s.ReleasesService.Create(app, config, slug, desc)
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
