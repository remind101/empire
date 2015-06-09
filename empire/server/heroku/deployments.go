package heroku

import (
	"fmt"
	"net/http"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	*empire.Empire
}

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image empire.Image
}

// Serve implements the Handler interface.
func (h *PostDeploys) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	var form PostDeployForm

	if err := Decode(req, &form); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; boundary=NL")

	var (
		r   *empire.Release
		err error
	)

	ch := make(chan empire.Event)
	errCh := make(chan error)
	go func() {
		r, err = h.DeployImage(ctx, form.Image, ch)
		errCh <- err
	}()

	for {
		select {
		case evt := <-ch:
			if err := Stream(w, evt); err != nil {
				Stream(w, newJSONMessageError(err))
				return nil
			}
			continue
		case err := <-errCh:
			if err != nil {
				Stream(w, newJSONMessageError(err))
				return nil
			}
		}

		break
	}

	Stream(w, &empire.DockerEvent{
		Status: fmt.Sprintf("Status: Created new release v%d for %s", r.Version, r.App.Name),
	})

	return nil
}

func newJSONMessageError(err error) jsonmessage.JSONMessage {
	return jsonmessage.JSONMessage{
		ErrorMessage: err.Error(),
		Error: &jsonmessage.JSONError{
			Message: err.Error(),
		},
	}
}
