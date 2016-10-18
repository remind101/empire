package main

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"text/template"
	"time"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor/client/conveyor"
	"github.com/remind101/empire"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
	githubauth "github.com/remind101/empire/server/auth/github"
	"github.com/remind101/empire/server/cloudformation"
	"github.com/remind101/empire/server/github"
	"github.com/remind101/empire/server/middleware"
	"github.com/remind101/empire/stats"
	"golang.org/x/oauth2"
)

func runServer(c *cli.Context) {
	ctx, err := newContext(c)
	if err != nil {
		log.Fatal(err)
	}

	// Send runtime metrics to stats backend.
	go stats.Runtime(ctx.Stats())

	port := c.String(FlagPort)

	if c.Bool(FlagAutoMigrate) {
		runMigrate(c)
	}

	db, err := newDB(ctx)
	if err != nil {
		log.Fatal(err)
	}

	e, err := newEmpire(db, ctx)
	if err != nil {
		log.Fatal(err)
	}

	// Do a preliminary health check to make sure everything is good at
	// boot.
	if err := e.IsHealthy(); err != nil {
		if err, ok := err.(*empire.IncompatibleSchemaError); ok {
			log.Fatal(fmt.Errorf("%v. You can resolve this error by running the migrations with `empire migrate` or with the `--automigrate` flag", err))
		}

		log.Fatal(err)
	} else {
		log.Println("Health checks passed")
	}

	if c.String(FlagCustomResourcesQueue) != "" {
		p := newCloudFormationCustomResourceProvisioner(e, ctx)
		log.Printf("Starting CloudFormation custom resource provisioner")
		go p.Start()
	}

	s := newServer(ctx, e)
	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}

func newServer(c *Context, e *empire.Empire) http.Handler {
	var opts server.Options
	opts.GitHub.Webhooks.Secret = c.String(FlagGithubWebhooksSecret)
	opts.GitHub.Deployments.Environments = strings.Split(c.String(FlagGithubDeploymentsEnvironments), ",")
	opts.GitHub.Deployments.ImageBuilder = newImageBuilder(c)
	opts.GitHub.Deployments.TugboatURL = c.String(FlagGithubDeploymentsTugboatURL)

	s := server.New(e, opts)
	s.Heroku.Auth = newAuth(c, e)
	s.Heroku.Secret = []byte(c.String(FlagSecret))

	h := middleware.Common(s)
	return middleware.Handler(c, h)
}

func newCloudFormationCustomResourceProvisioner(e *empire.Empire, c *Context) *cloudformation.CustomResourceProvisioner {
	p := cloudformation.NewCustomResourceProvisioner(e, c)
	p.QueueURL = c.String(FlagCustomResourcesQueue)
	p.Context = c
	return p
}

func newImageBuilder(c *Context) github.ImageBuilder {
	builder := c.String(FlagGithubDeploymentsImageBuilder)

	switch builder {
	case "template":
		tmpl := template.Must(template.New("image").Parse(c.String(FlagGithubDeploymentsImageTemplate)))
		return github.ImageFromTemplate(tmpl)
	case "conveyor":
		s := conveyor.NewService(conveyor.DefaultClient)
		s.URL = c.String(FlagConveyorURL)
		return github.NewConveyorImageBuilder(s)
	default:
		panic(fmt.Sprintf("unknown image builder: %s", builder))
	}
}

func newAuth(c *Context, e *empire.Empire) *auth.Auth {
	var (
		client         *githubauth.Client
		authenticators []auth.Authenticator
	)

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
			Scopes:       []string{"repo_deployment", "read:org", "user:email"},
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
		authenticator := githubauth.NewAuthenticator(client)
		if domain := c.String(FlagGithubIDDomain); domain != "" {
			authenticator.IDFunc = githubauth.EmailDomainID(domain)
		}
		authenticators = append(authenticators, authenticator)
	}

	// build the authentication chain.
	authenticator := auth.MultiAuthenticator(authenticators...)

	// After the user is authenticated, check their GitHub Organization membership.
	if org := c.String(FlagGithubOrg); org != "" {
		authorizer := githubauth.NewOrganizationAuthorizer(client)
		authorizer.Organization = org

		log.Println("Adding GitHub Organization authorizer with the following configuration:")
		log.Println(fmt.Sprintf("  Organization: %v ", org))

		return &auth.Auth{
			Authenticator: authenticator,
			// Cache the organization check for 30 minutes since
			// it's pretty slow.
			Authorizer: auth.CacheAuthorization(authorizer, 30*time.Minute),
		}
	}

	// After the user is authenticated, check their GitHub Team membership.
	if teamID := c.String(FlagGithubTeam); teamID != "" {
		authorizer := githubauth.NewTeamAuthorizer(client)
		authorizer.TeamID = teamID

		log.Println("Adding GitHub Team authorizer with the following configuration:")
		log.Println(fmt.Sprintf("  Team ID: %v ", teamID))

		return &auth.Auth{
			Authenticator: authenticator,
			// Cache the team check for 30 minutes
			Authorizer: auth.CacheAuthorization(authorizer, 30*time.Minute),
		}
	}

	return &auth.Auth{
		Authenticator: authenticator,
	}
}
