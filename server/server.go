// Package server provides an http.Handler implementation that includes the
// Heroku Platform API compatibility layer, GitHub Deployments integration and a
// simple health check.
package server

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"text/template"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/saml"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/empire/server/heroku"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
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
	}
}

// Server composes the Heroku API compatibility layer, the GitHub Webhooks
// handlers and a health check as a single http.Handler.
type Server struct {
	// Base host for the server.
	URL *url.URL

	// The underlying Heroku http.Handler.
	Heroku *heroku.Server

	// If provided, enables the SAML integration.
	ServiceProvider *saml.ServiceProvider

	mux *httpx.Router
}

func New(e *empire.Empire, options Options) *Server {
	r := httpx.NewRouter()
	s := &Server{mux: r}

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
	s.Heroku = heroku.New(e)
	r.Headers("Accept", heroku.AcceptHeader).Handler(s.Heroku)

	// Mount SAML handler.
	r.HandleFunc("/saml/acs", s.SAMLACS)

	// Mount health endpoint
	r.Handle("/health", NewHealthHandler(e))

	return s
}

func (s *Server) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return s.mux.ServeHTTPContext(ctx, w, r)
}

func (s *Server) SAMLACS(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if s.ServiceProvider == nil {
		http.NotFound(w, r)
		return nil
	}

	samlResponse := r.FormValue("SAMLResponse")
	assertion, err := s.ServiceProvider.ParseSAMLResponse(samlResponse, []string{""})
	if err != nil {
		if err, ok := err.(*saml.InvalidResponseError); ok {
			fmt.Fprintf(os.Stderr, "%v\n", err.PrivateErr)
		}
		//http.Error(w, "Unable to validate SAML Response", 403)
		http.Error(w, err.Error(), 403)
		return nil
	}

	// Create an Access Token for the API.
	login := assertion.Subject.NameID.Value
	at, err := s.Heroku.AccessTokensCreate(&heroku.AccessToken{
		User: &empire.User{
			Name: login,
		},
	})
	if err != nil {
		http.Error(w, err.Error(), 403)
		return nil
	}

	w.Header().Set("Content-Type", "text/html")
	instructionsTemplate.Execute(w, &instructionsData{
		URL:   s.URL,
		Login: login,
		Token: at.Token,
	})

	return nil
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

type instructionsData struct {
	URL   *url.URL
	Login string
	Token string
}

var instructionsTemplate = template.Must(template.New("instructions").Parse(`
<html>
<head>
<style>
pre.terminal {
  background-color: #444;
  color: #eee;
  padding: 20px;
  margin: 100px;
  overflow-x: scroll;
  border-radius: 4px;
}
</style>
</head>
<body>
<pre class="terminal">
<code>$ export EMPIRE_API_URL="{{.URL}}"
$ cat &lt;&lt;EOF &gt;&gt; ~/.netrc
machine {{.URL.Host}}
  login {{.Login}}
  password {{.Token}}
EOF</code>
</pre>
</body>
</html>
`))
