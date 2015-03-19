package server

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server/github"
	"github.com/remind101/empire/empire/server/heroku"
)

var DefaultOptions = Options{}

type Options struct {
	GitHub struct {
		ClientID     string
		ClientSecret string
		Organization string
		Secret       string
	}
}

func New(e *empire.Empire, options Options) http.Handler {
	r := mux.NewRouter()

	// TODO move authorizer out of package heroku.
	auth := heroku.NewAuthorizer(
		options.GitHub.ClientID,
		options.GitHub.ClientSecret,
		options.GitHub.Organization,
	)

	// Mount the heroku api
	h := heroku.New(e, auth)
	r.Headers("Accept", heroku.AcceptHeader).Handler(h)

	// Mount GitHub webhooks
	g := github.New(e, options.GitHub.Secret)
	r.MatcherFunc(githubWebhook).Handler(g)

	// Mount health endpoint
	r.Handle("/health", NewHealthHandler(e))

	return r
}

// githubWebhook is a mux.MatcherFunc that matches requests that have an
// `X-GitHub-Event` header present.
func githubWebhook(r *http.Request, rm *mux.RouteMatch) bool {
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

func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var status = http.StatusOK

	if !h.IsHealthy() {
		status = http.StatusServiceUnavailable
	}

	w.WriteHeader(status)
}
