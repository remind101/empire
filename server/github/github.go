package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"text/template"

	"github.com/ejholmes/hookshot"
	"github.com/ejholmes/hookshot/events"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/pkg/httpx"
	"golang.org/x/net/context"
)

var DefaultTemplate = `{{ .Repository.FullName }}:{{ .Deployment.Sha }}`

type Options struct {
	// The GitHub secret to ensure that the request was sent from GitHub.
	Secret string

	// If provided, specifies the environments that this Empire instance
	// should handle deployments for.
	Environments []string

	// ImageTemplate is used to determine the image to deploy.
	ImageTemplate string

	// TugboatURL can be provided if you want to send deployment logs to a
	// Tugboat instance.
	TugboatURL string
}

func New(e *empire.Empire, opts Options) httpx.Handler {
	r := hookshot.NewRouter()

	d := newDeployer(e, opts)
	secret := opts.Secret
	r.Handle("deployment", hookshot.Authorize(&DeploymentHandler{deployer: d, environments: opts.Environments}, secret))
	r.Handle("ping", hookshot.Authorize(http.HandlerFunc(Ping), secret))

	return r
}

// Deployment is an http.Handler for handling the `deployment` event.
type DeploymentHandler struct {
	deployer
	environments []string
}

func (h *DeploymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	panic("expected ServeHTTPContext to be called")
}

func (h *DeploymentHandler) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	var p events.Deployment

	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return nil
	}

	if !currentEnvironment(p.Deployment.Environment, h.environments) {
		w.WriteHeader(http.StatusNoContent)
		fmt.Fprintf(w, "Ignore deployment to environment: %s", p.Deployment.Environment)
		return nil
	}
	if err := h.deployer.Deploy(ctx, p, os.Stdout); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return nil
	}

	w.WriteHeader(http.StatusAccepted)
	io.WriteString(w, "Ok\n")
	return nil
}

func currentEnvironment(eventEnv string, environments []string) bool {
	for _, env := range environments {
		if env == eventEnv {
			return true
		}
	}
	return false
}

func Ping(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, "Ok\n")
}

// Image returns an image.Image for the given deployment.
func Image(t string, d events.Deployment) (image.Image, error) {
	if t == "" {
		t = DefaultTemplate
	}

	tmpl, err := template.New("image").Parse(t)
	if err != nil {
		return image.Image{}, err
	}

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, d); err != nil {
		return image.Image{}, err
	}

	return image.Decode(buf.String())
}
