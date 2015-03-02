package server

import (
	"net/http"

	"github.com/remind101/empire"
)

// PostDeploys is a Handler for the POST /v1/deploys endpoint.
type PostDeploys struct {
	DeploysService empire.DeploysService
}

// PostDeployForm is the form object that represents the POST body.
type PostDeployForm struct {
	Image struct {
		ID   string `json:"id"`
		Repo string `json:"repo"`
	} `json:"image"`
}

// Serve implements the Handler interface.
func (h *PostDeploys) Serve(req *Request) (int, interface{}, error) {
	var form PostDeployForm

	if err := req.Decode(&form); err != nil {
		return http.StatusInternalServerError, nil, err
	}

	d, err := h.DeploysService.Deploy(empire.Image{
		Repo: empire.Repo(form.Image.Repo),
		ID:   form.Image.ID,
	})
	if err != nil {
		return http.StatusInternalServerError, nil, err
	}

	return 201, d, nil
}
