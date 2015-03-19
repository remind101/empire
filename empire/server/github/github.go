package github

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/ejholmes/hookshot"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/server/middleware"
	"golang.org/x/net/context"
)

func New(e *empire.Empire, secret string) http.Handler {
	r := hookshot.NewRouter()

	opts := middleware.CommonOpts{
		Reporter: e.Reporter,
		ErrorHandler: func(err error, w http.ResponseWriter, r *http.Request) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		},
	}

	d := middleware.Common(&DeploymentHandler{e}, opts)
	p := middleware.Common(httpx.HandlerFunc(Ping), opts)

	r.Handle("deployment", hookshot.Authorize(d, secret))
	r.Handle("ping", hookshot.Authorize(p, secret))

	return r
}

// Deployment is the webhook payload for a deployment event.
type Deployment struct {
	Deployment struct {
		Sha         string `json:"sha"`
		Task        string `json:"task"`
		Environment string `json:"environment"`
	} `json:"deployment"`

	Repository struct {
		FullName string `json:"full_name"`
	} `json:"repository"`
}

// Deployment is an http.Handler for handling the `deployment` event.
type DeploymentHandler struct {
	*empire.Empire
}

func (h *DeploymentHandler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var p Deployment

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		return err
	}

	_, err := h.DeployCommit(empire.Commit{
		Repo: empire.Repo(p.Repository.FullName),
		Sha:  p.Deployment.Sha,
	})
	if err != nil {
		return err
	}

	io.WriteString(w, "Ok\n")
	return nil
}

func Ping(_ context.Context, w http.ResponseWriter, r *http.Request) error {
	io.WriteString(w, "Ok\n")
	return nil
}
