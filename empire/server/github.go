package server

import (
	"encoding/json"
	"net/http"

	"github.com/ejholmes/hookshot"
	"github.com/remind101/empire/empire"
)

// GitHubServer is an http.Handler for handling webhooks from GitHub.
type GitHubServer struct {
	http.Handler
}

func NewGitHubServer(e *empire.Empire) *GitHubServer {
	secret := e.Options.GitHub.Secret

	r := hookshot.NewRouter()
	r.Handle("deployment", hookshot.Authorize(&PostGitHubDeployment{e}, secret))

	return &GitHubServer{
		Handler: r,
	}
}

type PostGitHubDeployment struct {
	Empire
}

type GitHubDeploymentForm struct {
	Repo empire.Repo `json:"name"`
	Sha  string      `json:"sha"`
}

func (h *PostGitHubDeployment) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var form GitHubDeploymentForm

	if err := json.NewDecoder(r.Body).Decode(&form); err != nil {
		Error(w, err, http.StatusInternalServerError)
		return
	}

	d, err := h.DeployCommit(empire.Commit{
		Repo: form.Repo,
		Sha:  form.Sha,
	})
	if err != nil {
		Error(w, err, http.StatusInternalServerError)
		return
	}

	w.WriteHeader(200)
	Encode(w, d)
}
