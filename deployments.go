package empire

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

// deployerService is an implementation of the deployer interface that performs
// the core business logic to deploy.
type deployerService struct {
	*Empire
}

// createRelease creates a new release that can be deployed
func (s *deployerService) createRelease(ctx context.Context, db *gorm.DB, ss scheduler.StatusStream, opts DeployOpts) (*Release, error) {
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

func (s *deployerService) createInTransaction(ctx context.Context, stream scheduler.StatusStream, opts DeployOpts) (*Release, error) {
	tx := s.db.Begin()
	r, err := s.createRelease(ctx, tx, stream, opts)
	if err != nil {
		tx.Rollback()
		return r, err
	}
	return r, tx.Commit().Error
}

// Deploy is a thin wrapper around deploy to that adds the error to the
// jsonmessage stream.
func (s *deployerService) Deploy(ctx context.Context, opts DeployOpts) (*Release, error) {
	w := opts.Output

	var stream scheduler.StatusStream
	if opts.Stream {
		stream = w
	}

	r, err := s.createInTransaction(ctx, stream, opts)
	if err != nil {
		return r, w.Error(err)
	}

	if err := w.Status(fmt.Sprintf("Created new release v%d for %s", r.Version, r.App.Name)); err != nil {
		return r, err
	}

	if err := s.releases.Release(ctx, r, stream); err != nil {
		return r, w.Error(err)
	}

	return r, w.Status(fmt.Sprintf("Finished processing events for release v%d of %s", r.Version, r.App.Name))
}

// DeploymentStream provides a wrapper around an io.Writer for writing
// jsonmessage statuses, and implements the scheduler.StatusStream interface.
type DeploymentStream struct {
	w   io.Writer
	enc *json.Encoder
}

// NewDeploymentStream wraps the io.Writer as a DeploymentStream.
func NewDeploymentStream(w io.Writer) *DeploymentStream {
	return &DeploymentStream{
		w:   w,
		enc: json.NewEncoder(w),
	}
}

// Write implements the io.Writer interface. This allows things like the Docker
// daemon to write directly to the io.Writer, since it already writes in
// jsonmessage format.
func (w *DeploymentStream) Write(b []byte) (int, error) {
	return w.w.Write(b)
}

// Publish implements the scheduler.StatusStream interface.
func (w *DeploymentStream) Publish(status scheduler.Status) error {
	return w.Status(status.Message)
}

// Status writes a simple status update to the jsonmessage stream.
func (w *DeploymentStream) Status(message string) error {
	m := jsonmessage.JSONMessage{Status: fmt.Sprintf("Status: %s", message)}
	return w.encode(m)
}

// Error writes the error to the jsonmessage stream. The error that is provided
// is also returned, so that Error() can be used in return values.
func (w *DeploymentStream) Error(err error) error {
	if encErr := w.encode(newJSONMessageError(err)); encErr != nil {
		return encErr
	}
	return err
}

// encode encodes m into the stream.
func (w *DeploymentStream) encode(m jsonmessage.JSONMessage) error {
	return w.enc.Encode(m)
}

func newJSONMessageError(err error) jsonmessage.JSONMessage {
	return jsonmessage.JSONMessage{
		ErrorMessage: err.Error(),
		Error: &jsonmessage.JSONError{
			Message: err.Error(),
		},
	}
}
