package empire

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/pkg/jsonmessage"
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

// deployer is an interface that represents something that can perform a
// deployment.
type deployer interface {
	Deploy(context.Context, DeploymentsCreateOpts) (*Release, error)
}

// deployerService is an implementation of the deployer interface that performs
// the core business logic to deploy.
type deployerService struct {
	*appsService
	*configsService
	*slugsService
	*releasesService
}

// doDeploy does the actual deployment
func (s *deployerService) doDeploy(ctx context.Context, opts DeploymentsCreateOpts) (*Release, error) {
	app, img := opts.App, opts.Image

	// If no app is specified, attempt to find the app that relates to this
	// images repository, or create it if not found.
	if app == nil {
		var err error
		app, err = s.appsService.AppsFindOrCreateByRepo(img.Repository)
		if err != nil {
			return nil, err
		}
	} else {
		// If the app doesn't already have a repo attached to it, we'll attach
		// this image's repo.
		if err := s.appsService.AppsEnsureRepo(app, img.Repository); err != nil {
			return nil, err
		}
	}

	// Grab the latest config.
	config, err := s.ConfigsCurrent(app)
	if err != nil {
		return nil, err
	}

	// Create a new slug for the docker image.
	slug, err := s.SlugsCreateByImage(ctx, img, opts.Output)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	desc := fmt.Sprintf("Deploy %s", img.String())

	r, err := s.ReleasesCreate(ctx, &Release{
		App:         app,
		Config:      config,
		Slug:        slug,
		Description: desc,
	})

	return r, err
}

// Deploy is a thin wrapper around doDeploy to handle errors & output more cleanly
func (s *deployerService) Deploy(ctx context.Context, opts DeploymentsCreateOpts) (*Release, error) {
	var msg jsonmessage.JSONMessage

	r, err := s.doDeploy(ctx, opts)
	if err != nil {
		msg = newJSONMessageError(err)
	} else {
		msg = jsonmessage.JSONMessage{Status: fmt.Sprintf("Status: Created new release v%d for %s", r.Version, r.App.Name)}
	}

	if err := json.NewEncoder(opts.Output).Encode(&msg); err != nil {
		return r, err
	}

	return r, nil
}
