package main

import (
	"log"
	"net/http"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server"
)

func runServer(c *cli.Context) {
	port := c.String(FlagPort)

	if c.Bool(FlagAutoMigrate) {
		runMigrate(c)
	}

	e, err := newEmpire(c)
	if err != nil {
		log.Fatal(err)
	}

	s := newServer(c, e)
	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}

func newServer(c *cli.Context, e *empire.Empire) http.Handler {
	opts := server.Options{}
	opts.GitHub.ClientID = c.String(FlagGithubClient)
	opts.GitHub.ClientSecret = c.String(FlagGithubClientSecret)
	opts.GitHub.Organization = c.String(FlagGithubOrg)
	opts.GitHub.ApiURL = c.String(FlagGithubApiURL)
	opts.GitHub.Webhooks.Secret = c.String(FlagGithubWebhooksSecret)
	opts.GitHub.Deployments.Environment = c.String(FlagGithubDeploymentsEnvironment)
	opts.GitHub.Deployments.ImageTemplate = c.String(FlagGithubDeploymentsImageTemplate)
	opts.GitHub.Deployments.TugboatURL = c.String(FlagGithubDeploymentsTugboatURL)

	return server.New(e, opts)
}
