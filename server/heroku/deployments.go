package heroku

import (
	"net/http"

	"github.com/remind101/empire/pkg/image"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	*empire.Empire
}

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image image.Image
}

// Serve implements the Handler interface.
func (h *PostDeploys) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	var form PostDeployForm

	if err := Decode(req, &form); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "application/json; boundary=NL")

	if form.Image.Tag == "" && form.Image.Digest == "" {
		form.Image.Tag = "latest"
	}

	user, _ := empire.UserFromContext(ctx)

	h.Deploy(ctx, empire.DeployOpts{
		Image:  form.Image,
		Output: w,
		User:   user,
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
