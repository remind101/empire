package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

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
		log.Println("Using static authentication backend")
		// Fake authentication password where the user is "fake" and
		// password is blank.
		authenticators = append(authenticators, auth.StaticAuthenticator("fake", "", "", &empire.User{
			Name: "fake",
		}))
	} else {
		config := &oauth2.Config{
			ClientID:     c.String(FlagGithubClient),
			ClientSecret: c.String(FlagGithubClientSecret),
			Scopes:       []string{"repo_deployment", "read:org"},
		}

		client = github.NewClient(config)
		client.URL = c.String(FlagGithubApiURL)

		log.Println("Using GitHub authentication backend with the following configuration:")
		log.Println(fmt.Sprintf("  ClientID: %v", config.ClientID))
		log.Println(fmt.Sprintf("  ClientSecret: ****"))
		log.Println(fmt.Sprintf("  Scopes: %v", config.Scopes))
		log.Println(fmt.Sprintf("  GitHubAPI: %v", client.URL))

		// an authenticator for authenticating requests with a users github
		// credentials.
		authenticators = append(authenticators, github.NewAuthenticator(client))
	}

	// try access token before falling back to github.
	authenticator := auth.MultiAuthenticator(authenticators...)

	// After the user is authenticated, check their GitHub Organization membership.
	if org := c.String(FlagGithubOrg); org != "" {
		authorizer := github.NewOrganizationAuthorizer(client)
		authorizer.Organization = org

		log.Println("Adding GitHub Organization authorizer with the following configuration:")
		log.Println(fmt.Sprintf("  Organization: %v ", org))

		return auth.WithAuthorization(
			authenticator,
			// Cache the organization check for 30 minutes since
			// it's pretty slow.
			auth.CacheAuthorization(authorizer, 30*time.Minute),
		)
	}

	return authenticator
}
