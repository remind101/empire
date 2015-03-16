package heroku

import (
	"net/http"

	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	Empire
}

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image struct {
		Tag  string `json:"tag"`
		Repo string `json:"repo"`
	} `json:"image"`
}

// Serve implements the Handler interface.
func (h *PostDeploys) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var form PostDeployForm

	if err := Decode(r, &form); err != nil {
		return err
	}

	d, err := h.DeployImage(empire.Image{
		Repo: empire.Repo(form.Image.Repo),
		Tag:  form.Image.Tag,
	})
	if err != nil {
		return err
	}

	w.WriteHeader(201)
	return Encode(w, d)
}
