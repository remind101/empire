package server

import (
	"net/http"

	"github.com/codegangsta/negroni"
	"github.com/gorilla/mux"
	"github.com/kr/githubauth"
	"github.com/remind101/tugboat"
	"github.com/remind101/tugboat/frontend"
	"github.com/remind101/tugboat/pkg/pusherauth"
	"github.com/remind101/tugboat/server/api"
	"github.com/remind101/tugboat/server/github"
)

type Config struct {
	GitHub struct {
		Secret, ClientID, ClientSecret, Organization string
	}

	Pusher struct {
		Key, Secret string
	}

	// CookieSecret is a secret key that will be used to sign cookies.
	CookieSecret [32]byte
}

func New(tug *tugboat.Tugboat, config Config) http.Handler {
	r := mux.NewRouter()

	// auth is a function that can wrap an http.Handler with authentication.
	auth := newAuthenticator(config)

	// Mount GitHub webhooks
	g := github.New(tug, config.GitHub.Secret)
	r.MatcherFunc(githubWebhook).Handler(g)

	// Mount the API.
	a := api.New(tug, api.Config{
		Auth:   auth,
		Secret: config.GitHub.Secret,
	})
	r.Headers("Accept", api.AcceptHeader).Handler(a)

	// Pusher authentication.
	p := auth(&pusherauth.Handler{
		Key:    config.Pusher.Key,
		Secret: []byte(config.Pusher.Secret),
	})
	r.Handle("/pusher/auth", p)

	// Fallback to serving the frontend.
	f := frontend.New("")
	f.PusherKey = config.Pusher.Key
	r.NotFoundHandler = auth(f)

	n := negroni.Classic()
	n.UseHandler(r)

	return n
}

// githubWebhook is a mux.MatcherFunc that matches requests that have an
// `X-GitHub-Event` header present.
func githubWebhook(r *http.Request, rm *mux.RouteMatch) bool {
	h := r.Header[http.CanonicalHeaderKey("X-GitHub-Event")]
	return len(h) > 0
}

type authenticator func(http.Handler) http.Handler

func newAuthenticator(config Config) authenticator {
	switch {
	case config.GitHub.ClientID != "":
		return githubAuthenticator(config)
	default:
		return func(h http.Handler) http.Handler {
			return h
		}
	}
}

func githubAuthenticator(config Config) authenticator {
	key := config.CookieSecret
	keys := []*[32]byte{&key}

	return func(h http.Handler) http.Handler {
		return &githubauth.Handler{
			RequireOrg:   config.GitHub.Organization,
			Keys:         keys,
			ClientID:     config.GitHub.ClientID,
			ClientSecret: config.GitHub.ClientSecret,
			Handler:      h,
		}
	}
}
