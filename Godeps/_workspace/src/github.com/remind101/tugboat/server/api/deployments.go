package api

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/remind101/tugboat"
)

type Deployment struct {
	ID          string     `json:"id"`
	GitHubID    int64      `json:"github_id"`
	User        string     `json:"user"`
	Repo        string     `json:"repo"`
	Sha         string     `json:"sha"`
	Ref         string     `json:"ref"`
	Environment string     `json:"environment"`
	Status      string     `json:"status"`
	Output      string     `json:"output"`
	Error       string     `json:"error"`
	Provider    string     `json:"provider"`
	CreatedAt   time.Time  `json:"createdAt"` // always set
	StartedAt   *time.Time `json:"startedAt"` // nil until started
	CompletedAt *time.Time `json:"completedAt"`
}

func newDeployment(d *tugboat.Deployment) *Deployment {
	return &Deployment{
		ID:          d.ID,
		GitHubID:    d.GitHubID,
		User:        d.User,
		Repo:        d.Repo,
		Sha:         d.Sha,
		Ref:         d.Ref,
		Environment: d.Environment,
		Status:      d.Status.String(),
		Error:       d.Error,
		Provider:    d.Provider,
		CreatedAt:   d.CreatedAt,
		StartedAt:   d.StartedAt,
		CompletedAt: d.CompletedAt,
	}
}

func newDeployments(ds []*tugboat.Deployment) []*Deployment {
	deployments := make([]*Deployment, len(ds))

	for i := 0; i < len(ds); i++ {
		deployments[i] = newDeployment(ds[i])
	}

	return deployments
}

type GetDeploymentsHandler struct {
	tugboat *tugboat.Tugboat
}

func (h *GetDeploymentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d, err := h.tugboat.Deployments(tugboat.DeploymentsQuery{
		Limit: 20,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(newDeployments(d))
}

type GetDeploymentHandler struct {
	tugboat *tugboat.Tugboat
}

func (h *GetDeploymentHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d, err := h.tugboat.DeploymentsFind(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	out, err := h.tugboat.Logs(d)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	res := newDeployment(d)
	res.Output = out

	json.NewEncoder(w).Encode(res)
}

var errInvalidToken = errors.New("Token is not valid for deployment")

func authProvider(tug *tugboat.Tugboat, h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, _, _ := r.BasicAuth()
		if _, err := tug.TokensFind(token); err != nil {
			http.Error(w, "Provided token is not valid", http.StatusUnauthorized)
			return
		}

		h.ServeHTTP(w, r)
	})
}

type PostDeploymentsHandler struct {
	tugboat *tugboat.Tugboat
}

func (h *PostDeploymentsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var f tugboat.DeployOpts
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	d, err := h.tugboat.DeploymentsCreate(f)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(d)
}

type PostLogsHandler struct {
	tugboat *tugboat.Tugboat
}

func (h *PostLogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d, err := h.tugboat.DeploymentsFind(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := h.tugboat.WriteLogs(d, r.Body); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

type PostStatusHandler struct {
	tugboat *tugboat.Tugboat
}

func (h *PostStatusHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	d, err := h.tugboat.DeploymentsFind(mux.Vars(r)["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	var f tugboat.StatusUpdate
	if err := json.NewDecoder(r.Body).Decode(&f); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := h.tugboat.UpdateStatus(d, f); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}
