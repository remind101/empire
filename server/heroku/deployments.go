package heroku

import (
	"net/http"

	"github.com/remind101/empire/pkg/image"
	streamhttp "github.com/remind101/empire/pkg/stream/http"

	"github.com/remind101/empire"
	"golang.org/x/net/context"
)

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image  image.Image
	Stream bool
}

// ServeHTTPContext implements the Handler interface.
func (h *Server) PostDeploys(ctx context.Context, w http.ResponseWriter, req *http.Request) error {
	opts, err := newDeployOpts(ctx, w, req)
	if err != nil {
		return err
	}

	// We ignore errors here since this is a streaming endpoint,
	// and the error is handled in the response message
	_, _ = h.Deploy(ctx, *opts)
	return nil
}

func newDeployOpts(ctx context.Context, w http.ResponseWriter, req *http.Request) (*empire.DeployOpts, error) {
	var form PostDeployForm

	if err := Decode(req, &form); err != nil {
		return nil, err
	}

	m, err := findMessage(req)
	if err != nil {
		return nil, err
	}

	w.Header().Set("Content-Type", "application/json; boundary=NL")

	if form.Image.Tag == "" && form.Image.Digest == "" {
		form.Image.Tag = "latest"
	}

	opts := empire.DeployOpts{
		User:    UserFromContext(ctx),
		Image:   form.Image,
		Output:  empire.NewDeploymentStream(streamhttp.StreamingResponseWriter(w)),
		Message: m,
		Stream:  form.Stream,
	}
	return &opts, nil
}
