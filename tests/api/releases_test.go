package empire_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
)

func TestReleaseList(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustDeploy(t, c, DefaultImage)

	releases := mustReleaseList(t, c, "acme-inc")

	if len(releases) != 1 {
		t.Fatal("Expected a release")
	}

	if got, want := releases[0].Version, 1; got != want {
		t.Fatalf("Version => %v; want %v", got, want)
	}
}

func TestReleaseRollback(t *testing.T) {
	t.Skip("TODO: Not implemented yet")

	c, s := NewTestClient(t)
	defer s.Close()

	// Deploy twice
	d := mustDeploy(t, c, DefaultImage)
	mustDeploy(t, c, DefaultImage)

	// Rollback to the first deploy.
	mustReleaseRollback(t, c, "acme-inc", d.Release.ID)
}

func mustReleaseList(t testing.TB, c *heroku.Client, appName string) []heroku.Release {
	releases, err := c.ReleaseList(appName, nil)
	if err != nil {
		t.Fatal(err)
	}

	return releases
}

func mustReleaseRollback(t testing.TB, c *heroku.Client, appName string, version string) *heroku.Release {
	release, err := c.ReleaseRollback(appName, version)
	if err != nil {
		t.Fatal(err)
	}

	return release
}
