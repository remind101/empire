package github_test

import (
	"context"
	"flag"
	"net/http"
	"testing"

	"golang.org/x/oauth2"

	"github.com/remind101/empire"
	"github.com/remind101/empire/internal/ghinstallation"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/storage/github"
	"github.com/stretchr/testify/assert"
)

var (
	githubOwner    = flag.String("test.github.owner", "", "Owner of the repo")
	githubRepo     = flag.String("test.github.repo", "", "Repo to commit to")
	githubBasePath = flag.String("test.github.basepath", "apps/test", "Base path to commit to")
	githubRef      = flag.String("test.github.ref", "refs/heads/master", "Git ref to merge into")

	// You can specify either a hard coded access token
	githubAccessToken = flag.String("test.github.access_token", "", "GitHub access token to use to make authenticated commits")

	// Or authenticate through a github app
	githubAppID          = flag.Int("test.github.app_id", 0, "GitHub App ID")
	githubInstallationID = flag.Int("test.github.installation_id", 0, "GitHub Installation ID")
	githubPrivateKeyPath = flag.String("test.github.private_key", "", "Path to private key")
)

// Does an complete functional test against a real GitHub repo.
func TestStorage(t *testing.T) {
	s := github.NewStorage(newHTTPClient(t))
	s.Owner = *githubOwner
	s.Repo = *githubRepo
	s.BasePath = *githubBasePath
	s.Ref = *githubRef

	app := &empire.App{
		Name:    "acme-inc",
		Version: 2,
		Environment: map[string]string{
			"FOO": "bar",
		},
		Image: &image.Image{Repository: "remind101/acme-inc"},
		Formation: empire.Formation{
			"web": {
				Command: empire.MustParseCommand("bash"),
			},
		},
	}

	user := &empire.User{
		Name: "ejholmes",
	}

	event := empire.DeployEvent{
		BaseEvent: empire.NewBaseEvent(user, "Some message included at deploy time"),
		App:       "acme-inc",
		Image:     "remind101/acme-inc:latest",
	}

	_, err := s.ReleasesCreate(app, event)
	assert.NoError(t, err)

	apps, err := s.Apps(empire.AppsQuery{})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(apps))
	assert.Equal(t, "acme-inc", apps[0].Name)

	foundApp, err := s.AppsFind(empire.AppsQuery{Name: &app.Name})
	assert.NoError(t, err)
	assert.Equal(t, app.Image, foundApp.Image)
	assert.Equal(t, app.Formation, foundApp.Formation)
	assert.Equal(t, app.Environment, foundApp.Environment)
	assert.Equal(t, app, foundApp)

	releases, err := s.Releases(empire.ReleasesQuery{
		App: &empire.App{
			Name: "acme-inc",
		},
	})
	assert.NoError(t, err)
	assert.Equal(t, 1, len(releases))
	assert.Equal(t, "Deployed remind101/acme-inc:latest to acme-inc", releases[0].Description)
}

func newHTTPClient(t testing.TB) *http.Client {
	if *githubAccessToken != "" {
		ctx := context.Background()
		ts := oauth2.StaticTokenSource(
			&oauth2.Token{AccessToken: *githubAccessToken},
		)
		tc := oauth2.NewClient(ctx, ts)
		return tc
	} else if *githubAppID != 0 {
		itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, *githubAppID, *githubInstallationID, *githubPrivateKeyPath)
		if err != nil {
			t.Fatal(err)
		}

		return &http.Client{Transport: itr}
	} else {
		t.Skip()
		return nil
	}
}
