package empire

import (
	"fmt"
	"io"

	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

// Deployment statuses.
const (
	StatusPending = "pending"
	StatusFailed  = "failed"
	StatusSuccess = "success"
)

// DeploymentsCreateOpts represents options that can be passed when creating a
// new Deployment.
type DeploymentsCreateOpts struct {
	// App is the app that is being deployed to.
	App *App

	// Image is the image that's being deployed.
	Image image.Image

	// User the user that is triggering the deployment.
	User *User

	// Output is an io.Writer where deployment output and events will be
	// streamed in jsonmessage format.
	Output io.Writer
}

type deployer struct {
	*appsService
	*configsService
	*slugsService
	*releasesService
}

// DeploymentsDo performs the Deployment.
func (s *deployer) DeploymentsDo(ctx context.Context, opts DeploymentsCreateOpts) (*Release, error) {
	app, image := opts.App, opts.Image

	// Grab the latest config.
	config, err := s.ConfigsCurrent(app)
	if err != nil {
		return nil, err
	}

	// Create a new slug for the docker image.
	slug, err := s.SlugsCreateByImage(ctx, image, opts.Output)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	desc := fmt.Sprintf("Deploy %s", image.String())
	return s.ReleasesCreate(ctx, &Release{
		App:         app,
		Config:      config,
		Slug:        slug,
		Description: desc,
	})
}

func (s *deployer) DeployImageToApp(ctx context.Context, opts DeploymentsCreateOpts) (*Release, error) {
	if err := s.appsService.AppsEnsureRepo(opts.App, opts.Image.Repository); err != nil {
		return nil, err
	}

	return s.DeploymentsDo(ctx, opts)
}

// Deploy deploys an Image to the cluster.
func (s *deployer) DeployImage(ctx context.Context, opts DeploymentsCreateOpts) (*Release, error) {
	app, err := s.appsService.AppsFindOrCreateByRepo(opts.Image.Repository)
	if err != nil {
		return nil, err
	}
	opts.App = app
	return s.DeployImageToApp(ctx, opts)
}
