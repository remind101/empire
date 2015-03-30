package github

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/ejholmes/hookshot"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/httpx"
	"github.com/remind101/empire/empire/server/middleware"
	"golang.org/x/net/context"
)

// Timeout is how long to wait for a deploy to finish, before doing it in the
// background.
var Timeout = 10 * time.Second

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

	type result struct {
		deployment *empire.Deployment
		err        error
	}

	ch := make(chan *result)

	go func() {
		d, err := h.DeployCommit(ctx, empire.Commit{
			Repo: empire.Repo(p.Repository.FullName),
			Sha:  p.Deployment.Sha,
		})

		ch <- &result{
			deployment: d,
			err:        err,
		}
	}()

	select {
	case d := <-ch:
		if d.err != nil {
			return d.err
		}

		io.WriteString(w, "Deployed\n")
	case <-time.After(Timeout):
		fmt.Fprintf(w, "Deploy is taking longer than %v, performing in the background", Timeout)
	}

	return nil
}

func Ping(_ context.Context, w http.ResponseWriter, r *http.Request) error {
	io.WriteString(w, "Ok\n")
	return nil
}
