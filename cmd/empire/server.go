package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"text/template"
	"time"

	"github.com/codegangsta/cli"
	"github.com/remind101/conveyor/client/conveyor"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/saml"
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
	s.URL = c.URL(FlagURL)
	s.Heroku.Auth = newAuth(c, e)
	s.Heroku.Secret = []byte(c.String(FlagSecret))

	sp, err := newSAMLServiceProvider(c)
	if err != nil {
		panic(err)
	}

	s.ServiceProvider = sp

	h := middleware.Common(s)
	return middleware.Handler(c, h)
}

func newSAMLServiceProvider(c *Context) (*saml.ServiceProvider, error) {
	metadataLocation := c.String(FlagSAMLMetadata)
	if metadataLocation == "" {
		// No SAML
		return nil, nil
	}

	var metadataContent []byte
	if metadataURI, err := url.Parse(metadataLocation); err != nil {
		// Assume that we've been passed the XML document.
		metadataContent = []byte(metadataLocation)
	} else {
		metadataContent, err = fetchSAMLMetadata(metadataURI)
		if err != nil {
			return nil, fmt.Errorf("error fetching metadata from %s: %v", metadataLocation, err)
		}
	}

	baseURL := c.URL(FlagURL)

	var metadata saml.Metadata
	if err := xml.Unmarshal(metadataContent, &metadata); err != nil {
		return nil, fmt.Errorf("error parsing SAML metadata: %v", err)
	}

	return &saml.ServiceProvider{
		IDPMetadata: &metadata,
		MetadataURL: fmt.Sprintf("%s/saml/metadata", baseURL),
		AcsURL:      fmt.Sprintf("%s/saml/acs", baseURL),
	}, nil
}

func fetchSAMLMetadata(uri *url.URL) ([]byte, error) {
	// TODO: Support file://
	scheme := uri.Scheme
	switch scheme {
	case "https", "http":
		req, err := http.NewRequest("GET", uri.String(), nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", fmt.Sprintf("Empire (%s)", empire.Version))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		if resp.StatusCode/100 != 2 {
			return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
		}
		return ioutil.ReadAll(resp.Body)
	default:
		return nil, fmt.Errorf("not able to fetch metadata via %s", scheme)
	}
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
	authBackend := c.String(FlagServerAuth)

	// For backwards compatibility. If the auth backend is unspecified, but
	// a github client id is provided, assume the GitHub auth backend.
	if authBackend == "" {
		if c.String(FlagGithubClient) != "" {
			authBackend = "github"
		} else {
			authBackend = "fake"
		}
	}

	// If a GitHub client id is provided, we'll use GitHub as an
	// authentication backend. Otherwise, we'll just use a static username
	// and password backend.
	switch authBackend {
	case "fake":
		log.Println("Using static authentication backend")
		// Fake authentication password where the user is "fake" and
		// password is blank.
		return &auth.Auth{
			Authenticator: auth.StaticAuthenticator("fake", "", "", &empire.User{
				Name: "fake",
			}),
		}
	case "github":
		config := &oauth2.Config{
			ClientID:     c.String(FlagGithubClient),
			ClientSecret: c.String(FlagGithubClientSecret),
			Scopes:       []string{"repo_deployment", "read:org"},
		}

		client := githubauth.NewClient(config)
		client.URL = c.String(FlagGithubApiURL)

		log.Println("Using GitHub authentication backend with the following configuration:")
		log.Println(fmt.Sprintf("  ClientID: %v", config.ClientID))
		log.Println(fmt.Sprintf("  ClientSecret: ****"))
		log.Println(fmt.Sprintf("  Scopes: %v", config.Scopes))
		log.Println(fmt.Sprintf("  GitHubAPI: %v", client.URL))

		// an authenticator for authenticating requests with a users github
		// credentials.
		authenticator := githubauth.NewAuthenticator(client)

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
	case "saml":
		// When using the SAML authentication backend, access tokens are
		// created through the browser, so no need for an authenticator.
		return &auth.Auth{}
	default:
		panic("unreachable")
	}
}
