// Package server provides an http.Handler implementation that includes the
// Heroku Platform API compatibility layer, GitHub Deployments integration and a
// simple health check.
package server

import (
	"io"
	"net/http"
	"net/url"

	"github.com/remind101/empire"
	"github.com/remind101/empire/internal/saml"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/empire/server/heroku"
)

var (
	DefaultOptions = Options{}
)

type Options struct {
	OauthRedirectURL string
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

// Server composes the Heroku API compatibility layer, the GitHub Webhooks
// handlers and a health check as a single http.Handler.
type Server struct {
	// Base host for the server.
	URL *url.URL

	// The underlying Heroku http.Handler.
	Heroku *heroku.Server

	GitHubWebhooks http.Handler

	Health *HealthHandler

	// If provided, enables the SAML integration.
	ServiceProvider *saml.ServiceProvider

	OauthRedirectURL *url.URL
}

func New(e *empire.Empire, options Options) *Server {
	s := &Server{}

	if options.OauthRedirectURL != "" {
		parsedUrl, err := url.Parse(options.OauthRedirectURL)
		if err == nil {
			s.OauthRedirectURL = parsedUrl
		}
	}
	if options.GitHub.Webhooks.Secret != "" {
		// Mount GitHub webhooks
		s.GitHubWebhooks = github.New(e, github.Options{
			Secret:       options.GitHub.Webhooks.Secret,
			Environments: options.GitHub.Deployments.Environments,
			Deployer:     newDeployer(e, options),
		})
	}

	s.Heroku = heroku.New(e)
	s.Health = NewHealthHandler(e)

	return s
}

func (s *Server) Handler(r *http.Request) http.Handler {
	h := s.handler(r)
	if h == nil {
		h = http.NotFoundHandler()
	}
	return h
}

func (s *Server) redirectOauth(w http.ResponseWriter, req *http.Request) {
	// Shallow copy the existing URL
	newDest := req.URL
	// Replace the hostname with the provided hostname
	newDest.Host = s.OauthRedirectURL.Host
	// If we've specified a scheme, use that as well
	if s.OauthRedirectURL.Scheme != "" {
		newDest.Scheme = s.OauthRedirectURL.Scheme
	} else {
		newDest.Scheme = "https"
	}
	// Redirect the original request to the new location
	http.Redirect(w, req, newDest.String(), http.StatusTemporaryRedirect)
}

func (s *Server) handler(r *http.Request) http.Handler {
	if r.Header.Get("X-GitHub-Event") != "" {
		return s.GitHubWebhooks
	}

	// Route to Heroku API.
	if r.Header.Get("Accept") == heroku.AcceptHeader {
		return s.Heroku
	}

	if s.OauthRedirectURL != nil && r.URL.Path =="/oauth/exchange" {
		return http.HandlerFunc(s.redirectOauth)
	}

	switch r.URL.Path {
	case "/saml/login":
		return http.HandlerFunc(s.SAMLLogin)
	case "/saml/acs":
		return http.HandlerFunc(s.SAMLACS)
	case "/health":
		return s.Health
	}

	return nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h := s.Handler(r)
	h.ServeHTTP(w, r)
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

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	err := h.IsHealthy()
	if err == nil {
		w.WriteHeader(http.StatusOK)
		return
	}

	w.WriteHeader(http.StatusServiceUnavailable)
	io.WriteString(w, err.Error())
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

	// Perform the deployment within a go routine so we don't timeout
	// githubs webhook requests.
	d = github.DeployAsync(d)

	return d
}
