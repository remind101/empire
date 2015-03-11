package github

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ejholmes/hookshot"
	"github.com/remind101/empire/empire"
)

func New(e *empire.Empire, secret string) http.Handler {
	r := hookshot.NewRouter()

	r.Handle("deployment", hookshot.Authorize(&DeploymentHandler{e}, secret))
	r.Handle("ping", hookshot.Authorize(http.HandlerFunc(Ping), secret))

	return r
}

// Deployment is the webhook payload for a deployment event.
type Deployment struct {
	Deployment struct {
		Sha         string `json:"sha"`
		Task        string `json:"task"`
		Environment string `json:"environment"`
	} `json:"deployment"`

	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// Deployment is an http.Handler for handling the `deployment` event.
type DeploymentHandler struct {
	*empire.Empire
}

func (h *DeploymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var p Deployment

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	_, err := h.DeployCommit(empire.Commit{
		Repo: empire.Repo(p.Repository.FullName),
		Sha:  p.Deployment.Sha,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	io.WriteString(w, "Ok\n")
}

func Ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ok\n")
}
