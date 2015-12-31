package server

import (
	"net/http"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/empire/server/heroku"
	"github.com/remind101/empire/server/middleware"
	"github.com/remind101/empire/server/slack"
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
			Environment   string
			ImageTemplate string
			TugboatURL    string
		}
	}
}

func New(e *empire.Empire, options Options) http.Handler {
	r := httpx.NewRouter()

	if options.GitHub.Webhooks.Secret != "" {
		// Mount GitHub webhooks
		g := github.New(e, github.Options{
			Secret:        options.GitHub.Webhooks.Secret,
			Environment:   options.GitHub.Deployments.Environment,
			ImageTemplate: options.GitHub.Deployments.ImageTemplate,
			TugboatURL:    options.GitHub.Deployments.TugboatURL,
		})
		r.Match(githubWebhook, g)
	}

	// Mount the heroku api
	h := heroku.New(e, options.Authenticator)
	r.Headers("Accept", heroku.AcceptHeader).Handler(h)

	// Mount health endpoint
	r.Handle("/health", NewHealthHandler(e))

	return middleware.Common(r, middleware.CommonOpts{
		Reporter: e.Reporter,
		Logger:   e.Logger,
	})
}

func NewSlack(e *empire.Empire, token string) http.Handler {
	s := slack.NewServer(e, token)
	return middleware.Common(s, middleware.CommonOpts{
		Reporter: e.Reporter,
		Logger:   e.Logger,
	})
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
	IsHealthy func() bool
}

// NewHealthHandler returns a new HealthHandler using the IsHealthy method from
// an Empire instance.
func NewHealthHandler(e *empire.Empire) *HealthHandler {
	return &HealthHandler{
		IsHealthy: e.IsHealthy,
	}
}

func (h *HealthHandler) ServeHTTPContext(_ context.Context, w http.ResponseWriter, r *http.Request) error {
	var status = http.StatusOK

	if !h.IsHealthy() {
		status = http.StatusServiceUnavailable
	}

	w.WriteHeader(status)

	return nil
}
