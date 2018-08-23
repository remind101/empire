package github_test

import (
	"flag"
	"net/http"
	"testing"

	"github.com/remind101/empire"
	"github.com/remind101/empire/internal/ghinstallation"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/storage/github"
	"github.com/stretchr/testify/assert"
)

var (
	githubAppID          = flag.Int("test.github.app_id", 0, "GitHub App ID")
	githubInstallationID = flag.Int("test.github.installation_id", 0, "GitHub Installation ID")
	githubPrivateKeyPath = flag.String("test.github.private_key", "", "Path to private key")

	githubOwner    = flag.String("test.github.owner", "", "Owner of the repo")
	githubRepo     = flag.String("test.github.repo", "", "Repo to commit to")
	githubBasePath = flag.String("test.github.basepath", "apps/test", "Base path to commit to")
	githubRef      = flag.String("test.github.ref", "refs/heads/master", "Git ref to merge into")
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

	_, err := s.ReleasesCreate(app, "Deploying new image")
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
	assert.Equal(t, "Deploying new image", releases[0].Description)
}

func newHTTPClient(t testing.TB) *http.Client {
	if *githubAppID == 0 {
		t.Skip()
	}

	itr, err := ghinstallation.NewKeyFromFile(http.DefaultTransport, *githubAppID, *githubInstallationID, *githubPrivateKeyPath)
	if err != nil {
		t.Fatal(err)
	}

	return &http.Client{Transport: itr}
}
