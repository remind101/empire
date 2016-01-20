package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
	githubauth "github.com/remind101/empire/server/auth/github"
	"github.com/remind101/empire/server/github"
	"golang.org/x/oauth2"
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
	opts.GitHub.Deployments.Environments = strings.Split(c.String(FlagGithubDeploymentsEnvironments), ",")
	opts.GitHub.Deployments.ImageBuilder = newImageBuilder(c)
	opts.GitHub.Deployments.TugboatURL = c.String(FlagGithubDeploymentsTugboatURL)

	return server.New(e, opts)
}

func newImageBuilder(c *cli.Context) github.ImageBuilder {
	tmpl := template.Must(template.New("image").Parse(c.String(FlagGithubDeploymentsImageTemplate)))
	return github.ImageFromTemplate(tmpl)
}

func newAuthenticator(c *cli.Context, e *empire.Empire) auth.Authenticator {
	// an authenticator authenticating requests with a users empire acccess
	// token.
	authenticators := []auth.Authenticator{
		auth.NewAccessTokenAuthenticator(e),
	}

	var client *githubauth.Client
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

		client = githubauth.NewClient(config)
		client.URL = c.String(FlagGithubApiURL)

		log.Println("Using GitHub authentication backend with the following configuration:")
		log.Println(fmt.Sprintf("  ClientID: %v", config.ClientID))
		log.Println(fmt.Sprintf("  ClientSecret: ****"))
		log.Println(fmt.Sprintf("  Scopes: %v", config.Scopes))
		log.Println(fmt.Sprintf("  GitHubAPI: %v", client.URL))

		// an authenticator for authenticating requests with a users github
		// credentials.
		authenticators = append(authenticators, githubauth.NewAuthenticator(client))
	}

	// try access token before falling back to github.
	authenticator := auth.MultiAuthenticator(authenticators...)

	// After the user is authenticated, check their GitHub Organization membership.
	if org := c.String(FlagGithubOrg); org != "" {
		authorizer := githubauth.NewOrganizationAuthorizer(client)
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
