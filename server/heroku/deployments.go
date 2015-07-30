package heroku

import (
	"net/http"

	"github.com/remind101/empire/pkg/image"
	streamhttp "github.com/remind101/empire/pkg/stream/http"

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
	opts, err := newDeploymentsCreateOpts(ctx, w, req)
	if err != nil {
		return err
	}
	h.Deploy(ctx, *opts)
	return nil
}

func newDeploymentsCreateOpts(ctx context.Context, w http.ResponseWriter, req *http.Request) (*empire.DeploymentsCreateOpts, error) {
	var form PostDeployForm

	if err := Decode(req, &form); err != nil {
		return nil, err
	}

	w.Header().Set("Content-Type", "application/json; boundary=NL")

	if form.Image.Tag == "" && form.Image.Digest == "" {
		form.Image.Tag = "latest"
	}

	user, _ := empire.UserFromContext(ctx)

	opts := empire.DeploymentsCreateOpts{
		Image:  form.Image,
		Output: streamhttp.StreamingResponseWriter(w),
		User:   user,
	}
	return &opts, nil
}
