package server

import (
	"net/http"

	"github.com/remind101/empire/empire"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/empire/empire/server/authorization"
	githubauth "github.com/remind101/empire/empire/server/authorization/github"
	"github.com/remind101/empire/empire/server/heroku"
	"github.com/remind101/empire/empire/server/middleware"
	"golang.org/x/net/context"
)

var (
	DefaultOptions = Options{}

	// DefaultGitHubScopes is the default oauth scopes to obtain when getting an
	// authorization from GitHub.
	DefaultGitHubScopes = []string{
		"repo_deployment", // For creating deployment statuses.
		"read:org",        // For reading organization memberships.
	}
)

type Options struct {
	GitHub struct {
		ClientID     string
		ClientSecret string
		Organization string
	}
}

func New(e *empire.Empire, options Options) http.Handler {
	r := httpx.NewRouter()

	auth := NewAuthorizer(
		options.GitHub.ClientID,
		options.GitHub.ClientSecret,
		options.GitHub.Organization,
	)

	// Mount the heroku api
	h := heroku.New(e, auth)
	r.Header("Accept", heroku.AcceptHeader, h)

	// Mount health endpoint
	r.Handle("GET", "/health", NewHealthHandler(e))

	return middleware.Common(r, middleware.CommonOpts{
		Reporter: e.Reporter,
	})
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

// NewAuthorizer returns a new Authorizer. If the client id is present, it will
// return a real Authorizer that talks to GitHub. If an empty string is
// provided, then it will just return a fake authorizer.
func NewAuthorizer(clientID, clientSecret, organization string) authorization.Authorizer {
	if clientID == "" {
		return &authorization.Fake{}
	}

	return &githubauth.Authorizer{
		Scopes:       DefaultGitHubScopes,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Organization: organization,
	}
}
