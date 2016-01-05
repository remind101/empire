package api

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/remind101/tugboat"
)

const AcceptHeader = "application/vnd.tugboat+json; version=1"

type Config struct {
	Auth   func(http.Handler) http.Handler
	Secret string
}

func New(t *tugboat.Tugboat, config Config) http.Handler {
	r := mux.NewRouter()

	auth := config.Auth

	r.Handle("/jobs", auth(&GetDeploymentsHandler{t})).Methods("GET")
	r.Handle("/jobs/{id}", auth(&GetDeploymentHandler{t})).Methods("GET")
	r.Handle("/deployments", authProvider(t, &PostDeploymentsHandler{t})).Methods("POST")
	r.Handle("/deployments/{id}/logs", authProvider(t, &PostLogsHandler{t})).Methods("POST")
	r.Handle("/deployments/{id}/status", authProvider(t, &PostStatusHandler{t})).Methods("POST")

	return r
}
