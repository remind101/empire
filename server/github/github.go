// Package github provides an http.Handler implementation that allows Empire to
// handle GitHub Deployments.
package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
)

var (
	// DefaultTemplate is a text/template string that will be used to map a
	// deployment event to a docker image to deploy.
	DefaultTemplate = `{{ .Repository.FullName }}:{{ .Deployment.Sha }}`
)

type Options struct {
	// The GitHub secret to ensure that the request was sent from GitHub.
	Secret string

	// If provided, specifies the environments that this Empire instance
	// should handle deployments for.
	Environments []string

	Deployer Deployer
}

func New(e *empire.Empire, opts Options) http.Handler {
	r := hookshot.NewRouter()

	secret := opts.Secret
	r.Handle("deployment", hookshot.Authorize(&DeploymentHandler{Deployer: opts.Deployer, environments: opts.Environments}, secret))
	r.Handle("ping", hookshot.Authorize(http.HandlerFunc(Ping), secret))

	return r
}

// Deployment is an http.Handler for handling the `deployment` event.
type DeploymentHandler struct {
	Deployer
	environments []string
}

func (h *DeploymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	var p events.Deployment

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if !currentEnvironment(p.Deployment.Environment, h.environments) {
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "Ignore deployment to environment: %s", p.Deployment.Environment)
		return
	}
	if err := h.Deploy(ctx, p, os.Stdout); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, "Ok\n")
	return
}

func currentEnvironment(eventEnv string, environments []string) bool {
	for _, env := range environments {
		if env == eventEnv {
			return true
		}
	}
	return false
}

func Ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ok\n")
}
