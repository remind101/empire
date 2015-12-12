package main

import (
	"log"
	"net/http"

	"golang.org/x/oauth2"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/auth/github"
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
	var opts server.Options
	opts.Authenticator = newAuthenticator(c, e)
	opts.GitHub.Webhooks.Secret = c.String(FlagGithubWebhooksSecret)
	opts.GitHub.Deployments.Environment = c.String(FlagGithubDeploymentsEnvironment)
	opts.GitHub.Deployments.ImageTemplate = c.String(FlagGithubDeploymentsImageTemplate)
	opts.GitHub.Deployments.TugboatURL = c.String(FlagGithubDeploymentsTugboatURL)

	return server.New(e, opts)
}

func newAuthenticator(c *cli.Context, e *empire.Empire) auth.Authenticator {
	// an authenticator authenticating requests with a users empire acccess
	// token.
	authenticators := []auth.Authenticator{
		auth.NewAccessTokenAuthenticator(e),
	}

	var client *github.Client
	// If a GitHub client id is provided, we'll use GitHub as an
	// authentication backend. Otherwise, we'll just use a static username
	// and password backend.
	if c.String(FlagGithubClient) == "" {
		// Fake authentication password where the user is "fake" and
		// password is blank.
		authenticators = append(authenticators, auth.StaticAuthenticator("fake", "", "", &empire.User{
			Name: "fake",
		}))
	} else {
		client = github.NewClient(&oauth2.Config{
			ClientID:     c.String(FlagGithubClient),
			ClientSecret: c.String(FlagGithubClientSecret),
			Scopes:       []string{"repo_deployment", "read:org"},
		})
		client.URL = c.String(FlagGithubApiURL)

		// an authenticator for authenticating requests with a users github
		// credentials.
		authenticators = append(authenticators, github.NewAuthenticator(client))
	}

	// try access token before falling back to github.
	authenticator := auth.MultiAuthenticator(authenticators...)

	if org := c.String(FlagGithubOrg); org != "" {
		authorizer := github.NewOrganizationAuthorizer(client)
		authorizer.Organization = org
		authenticator = auth.WithAuthorization(authenticator, authorizer)
	}

	return authenticator
}
