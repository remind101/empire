package github

import (
	"fmt"
	"io"
	"net/http"

	"github.com/ejholmes/hookshot"
	"github.com/remind101/tugboat"
	"golang.org/x/net/context"
)

func New(tug *tugboat.Tugboat, secret string) http.Handler {
	r := hookshot.NewRouter()

	r.Handle("ping", http.HandlerFunc(Ping))
	r.Handle("deployment", hookshot.Authorize(&DeploymentHandler{tugboat: tug}, secret))

	return r
}

func Ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ok\n")
}

type DeploymentHandler struct {
	tugboat *tugboat.Tugboat
}

func (h *DeploymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	opts, err := tugboat.NewDeployOptsFromReader(r.Body)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	ds, err := h.tugboat.Deploy(context.TODO(), opts)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	for _, d := range ds {
		fmt.Fprintf(w, "Deployment: %s\n", d.ID)
	}
}
