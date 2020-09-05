// Package server provides an http.Handler implementation that includes the
// Heroku Platform API compatibility layer, GitHub Deployments integration and a
// simple health check.
package server

import (
	"fmt"
	"golang.org/x/oauth2"
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
		OAuth struct {
			ClientID string
			ClientSecret string
			RedirectURL string
			Scopes []string
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

	AuthConfig *oauth2.Config
}

func New(e *empire.Empire, options Options) *Server {
	s := &Server{}

	if options.GitHub.Webhooks.Secret != "" {
		// Mount GitHub webhooks
		s.GitHubWebhooks = github.New(e, github.Options{
			Secret:       options.GitHub.Webhooks.Secret,
			Environments: options.GitHub.Deployments.Environments,
			Deployer:     newDeployer(e, options),
		})
	}

	if options.GitHub.OAuth.ClientID != "" {
		s.AuthConfig = &oauth2.Config{
			ClientID: options.GitHub.OAuth.ClientID,
			ClientSecret: options.GitHub.OAuth.ClientSecret,
			Scopes: options.GitHub.OAuth.Scopes,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://github.com/login/oauth/authorize",
				TokenURL: "https://github.com/login/oauth/access_token",
			},
			RedirectURL: options.GitHub.OAuth.RedirectURL + "/oauth/exchange",
		}
	}

	s.Heroku = heroku.New(e)
	s.Health = NewHealthHandler(e)

	return s
}

func (s *Server) StartWebFlow(w http.ResponseWriter, r *http.Request) {
	// Construct a URL for the initial webflow request
	url := s.AuthConfig.AuthCodeURL(fmt.Sprintf(r.FormValue("port")))
	// Issue a redirect
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
}

func reportOAuthFailure(w http.ResponseWriter, r *http.Request, port string, err ...interface{}) {
	query := url.Values{}
	query.Add("message", fmt.Sprint(err...))
	redirectUrl := url.URL{
		Host: fmt.Sprintf("localhost:%s",port),
		RawQuery: query.Encode(),
		Path: "/oauth/failure",
	}
	http.Redirect(w, r, redirectUrl.String(), http.StatusTemporaryRedirect)
}

func (s *Server) PerformCodeExchange(w http.ResponseWriter, r *http.Request) {
	token, err := s.AuthConfig.Exchange(oauth2.NoContext, r.FormValue("code"))
	port := r.FormValue("state")
	if err != nil {
		reportOAuthFailure(w, r, port, err)
		return
	}

	if token == nil || token.AccessToken == "" {
		reportOAuthFailure(w, r, port, "Could not retrieve auth token")
		return
	}

	// We COULD directly generate and return the Signed JWT to the client here and return it to the client, but it kind of feels out
	// of place.  We've opted instead for re-using as much of the existing authentication path as possible
	// It's worth thinking about the security implications here.  The JWT is signed and can't be modified, and has an expiration date in
	// the near future.  But the Github token we're sending back typically has a lifetime of a year.  This is mitigated by the fact
	// that the Github token we're sending back is being sent to the browser is over https as a redirect response.  The browser then makes
	// an unencrypted http request, but only to localhost, so an attacker would have to be able to capture packets on the loopback interface
	// to get it, which in turn requires root access, so the entire client machine would already be compromised at that point.
	redirectUrl := fmt.Sprintf("http://localhost:%s/oauth/token?token=%s", r.FormValue("state"), url.QueryEscape(token.AccessToken))
	http.Redirect(w, r, redirectUrl, http.StatusTemporaryRedirect)
}

func (s *Server) Handler(r *http.Request) http.Handler {
	h := s.handler(r)
	if h == nil {
		h = http.NotFoundHandler()
	}
	return h
}

func (s *Server) handler(r *http.Request) http.Handler {
	if r.Header.Get("X-GitHub-Event") != "" {
		return s.GitHubWebhooks
	}

	// Route to Heroku API.
	if r.Header.Get("Accept") == heroku.AcceptHeader {
		return s.Heroku
	}

	switch r.URL.Path {
	case "/saml/login":
		return http.HandlerFunc(s.SAMLLogin)
	case "/saml/acs":
		return http.HandlerFunc(s.SAMLACS)
	case "/health":
		return s.Health

	// These endpoints get hit by clients using the browser in order to do the web-flow
	// version of authentication
	case "/oauth/start":
		return http.HandlerFunc(s.StartWebFlow)
	case "/oauth/exchange":
		return http.HandlerFunc(s.PerformCodeExchange)
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
