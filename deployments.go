package empire

import (
	"encoding/json"
	"fmt"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/jinzhu/gorm"
	"golang.org/x/net/context"
)

// deployerService is an implementation of the deployer interface that performs
// the core business logic to deploy.
type deployerService struct {
	*Empire
}

// deploy does the actual deployment
func (s *deployerService) deploy(ctx context.Context, db *gorm.DB, opts DeploymentsCreateOpts) (*Release, error) {
	app, img := opts.App, opts.Image

	// If no app is specified, attempt to find the app that relates to this
	// images repository, or create it if not found.
	if app == nil {
		var err error
		app, err = appsFindOrCreateByRepo(db, img.Repository)
		if err != nil {
			return nil, err
		}
	} else {
		// If the app doesn't already have a repo attached to it, we'll attach
		// this image's repo.
		if err := appsEnsureRepo(db, app, img.Repository); err != nil {
			return nil, err
		}
	}

	// Grab the latest config.
	config, err := s.configs.Config(db, app)
	if err != nil {
		return nil, err
	}

	// Create a new slug for the docker image.
	slug, err := s.slugs.Create(ctx, db, img, opts.Output)
	if err != nil {
		return nil, err
	}

	// Create a new release for the Config
	// and Slug.
	desc := fmt.Sprintf("Deploy %s", img.String())
	desc = appendMessageToDescription(desc, opts.User, opts.Message)

	r, err := s.releases.Create(ctx, db, &Release{
		App:         app,
		Config:      config,
		Slug:        slug,
		Description: desc,
	})

	return r, err
}

// Deploy is a thin wrapper around deploy to that adds the error to the
// jsonmessage stream.
func (s *deployerService) Deploy(ctx context.Context, db *gorm.DB, opts DeploymentsCreateOpts) (*Release, error) {
	var msg jsonmessage.JSONMessage

	r, err := s.deploy(ctx, db, opts)
	if err != nil {
		msg = newJSONMessageError(err)
	} else {
		msg = jsonmessage.JSONMessage{Status: fmt.Sprintf("Status: Created new release v%d for %s", r.Version, r.App.Name)}
	}

	if err := json.NewEncoder(opts.Output).Encode(&msg); err != nil {
		return r, err
	}

	return r, err
}
