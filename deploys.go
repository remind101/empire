package empire

import (
	"github.com/remind101/empire/deploys"
	"github.com/remind101/empire/images"
)

// DeploysService is an interface that can be implemented to deploy images.
type DeploysService interface {
	// Deploy deploys a container image to the cluster.
	Deploy(*images.Image) (*deploys.Deploy, error)
}

// NewDeploysService returns a new DeploysService.
func NewDeploysService(options Options, apps AppsService, configs ConfigsService, slugs SlugsService, releases ReleasesService, manager Manager) (DeploysService, error) {
	return &deploysService{
		AppsService:     apps,
		ConfigsService:  configs,
		Manager:         manager,
		SlugsService:    slugs,
		ReleasesService: releases,
	}, nil
}

// deploysService is a base implementation of the DeploysService
type deploysService struct {
	AppsService
	ConfigsService
	SlugsService
	ReleasesService
	Manager
}

// Deploy deploys an Image to the cluster.
func (s *deploysService) Deploy(image *images.Image) (*deploys.Deploy, error) {
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
	release, err := s.ReleasesService.Create(app, config, slug)
	if err != nil {
		return nil, err
	}

	// Schedule the new release onto the cluster.
	if err := s.Manager.ScheduleRelease(release); err != nil {
		return nil, err
	}

	// We're deployed! ...
	// hopefully.
	return &deploys.Deploy{
		ID:      "1",
		Release: release,
	}, nil
}
