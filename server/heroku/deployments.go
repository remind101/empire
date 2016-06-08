package heroku

import (
	"net/http"

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
	Image string
}

// ServeHTTPContext implements the Handler interface.
func (h *PostDeploys) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	opts, err := newDeploymentsCreateOpts(ctx, w, req)
	if err != nil {
		return err
	}

	// We ignore errors here since this is a streaming endpoint,
	// and the error is handled in the response message
	_, _ = h.Deploy(ctx, *opts)
	return nil
}

func newDeploymentsCreateOpts(ctx context.Context, w http.ResponseWriter, req *http.Request) (*empire.DeploymentsCreateOpts, error) {
	var form PostDeployForm

	if err := Decode(req, &form); err != nil {
		return nil, err
	}

	m, err := findMessage(req)
	if err != nil {
		return nil, err
	}

	w.Header().Set("Content-Type", "application/json; boundary=NL")

	opts := empire.DeploymentsCreateOpts{
		User:    UserFromContext(ctx),
		Image:   form.Image,
		Output:  streamhttp.StreamingResponseWriter(w),
		Message: m,
	}
	return &opts, nil
}
