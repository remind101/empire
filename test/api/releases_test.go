package api_test

import (
	"testing"

	"github.com/remind101/empire/pkg/heroku"
)

func TestReleaseList(t *testing.T) {
	c := newClient(t)
	defer c.Close()

	mustDeploy(t, c.Client, DefaultImage)

	releases := mustReleaseList(t, c.Client, "acme-inc")

	if len(releases) != 1 {
		t.Fatal("Expected a release")
	}

	if got, want := releases[0].Version, 1; got != want {
		t.Fatalf("Version => %v; want %v", got, want)
	}
}

func TestReleaseInfo(t *testing.T) {
	c := newClient(t)
	defer c.Close()

	mustDeploy(t, c.Client, DefaultImage)

	release := mustReleaseInfo(t, c.Client, "acme-inc", "1")

	if got, want := release.Version, 1; got != want {
		t.Fatalf("Version => %v; want %v", got, want)
	}
}

func TestReleaseRollback(t *testing.T) {
	c := newClient(t)
	defer c.Close()

	// Deploy twice
	mustDeploy(t, c.Client, DefaultImage)
	mustDeploy(t, c.Client, DefaultImage)

	// Rollback to the first deploy.
	mustReleaseRollback(t, c.Client, "acme-inc", "1")
}

func mustReleaseList(t testing.TB, c *heroku.Client, appName string) []heroku.Release {
	releases, err := c.ReleaseList(appName, nil)
	if err != nil {
		t.Fatal(err)
	}

	return releases
}

func mustReleaseInfo(t testing.TB, c *heroku.Client, appName string, version string) *heroku.Release {
	release, err := c.ReleaseInfo(appName, version)
	if err != nil {
		t.Fatal(err)
	}

	return release
}

func mustReleaseRollback(t testing.TB, c *heroku.Client, appName string, version string) *heroku.Release {
	release, err := c.ReleaseRollback(appName, version, "")
	if err != nil {
		t.Fatal(err)
	}

	return release
}
