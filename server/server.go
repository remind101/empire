// Package server provides an http.Handler implementation that includes the
// Heroku Platform API compatibility layer, GitHub Deployments integration and a
// simple health check.
package server

import (
	"io"
	"net/http"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/empire/server/heroku"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

var (
	DefaultOptions = Options{}
)

type Options struct {
	Authenticator auth.Authenticator

	GitHub struct {
		// Deployments
		Webhooks struct {
			Secret string
		}
		Deployments struct {
			Environments []string
			ImageBuilder github.ImageBuilder
			TugboatURL   string
		}
	}
}

func New(e *empire.Empire, options Options) httpx.Handler {
	r := httpx.NewRouter()

	if options.GitHub.Webhooks.Secret != "" {
		// Mount GitHub webhooks
		g := github.New(e, github.Options{
			Secret:       options.GitHub.Webhooks.Secret,
			Environments: options.GitHub.Deployments.Environments,
			Deployer:     newDeployer(e, options),
		})
		r.Match(githubWebhook, g)
	}

	// Mount the heroku api
	hk := heroku.New(e)
	hk.Authenticator = options.Authenticator
	r.Headers("Accept", heroku.AcceptHeader).Handler(hk)

	// Mount health endpoint
	r.Handle("/health", NewHealthHandler(e))

	return r
}

// githubWebhook is a MatcherFunc that matches requests that have an
// `X-GitHub-Event` header present.
func githubWebhook(r *http.Request) bool {
	h := r.Header[http.CanonicalHeaderKey("X-GitHub-Event")]
	return len(h) > 0
}

// HealthHandler is an http.Handler that returns the health of empire.
type HealthHandler struct {
	// A function that returns true if empire is healthy.
	IsHealthy func() error
}

// NewHealthHandler returns a new HealthHandler using the IsHealthy method from
// an Empire instance.
func NewHealthHandler(e *empire.Empire) *HealthHandler {
	return &HealthHandler{
		IsHealthy: e.IsHealthy,
	}
}

func (h *HealthHandler) ServeHTTPContext(_ context.Context, w http.ResponseWriter, r *http.Request) error {
	err := h.IsHealthy()
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return nil
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	io.WriteString(w, err.Error())

	return nil
}

// newDeployer generates a new github.Deployer implementation for the given
// options.
func newDeployer(e *empire.Empire, options Options) github.Deployer {
	ed := github.NewEmpireDeployer(e)
	ed.ImageBuilder = options.GitHub.Deployments.ImageBuilder

	var d github.Deployer = ed

	// Enables the Tugboat integration, which will send logs to a Tugboat
	// instance.
	if url := options.GitHub.Deployments.TugboatURL; url != "" {
		d = github.NotifyTugboat(d, url)
	}

	// Add tracing information so we know about errors.
	d = github.TraceDeploy(d)

	// Perform the deployment within a go routine so we don't timeout
	// githubs webhook requests.
	d = github.DeployAsync(d)

	return d
}
