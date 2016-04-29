package api_test

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/heroku"
	"github.com/stretchr/testify/assert"
)

func TestAppCreate(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	app := mustAppCreate(t, c, empire.App{
		Name: "acme-inc",
	})

	if got, want := app.Name, "acme-inc"; got != want {
		t.Fatalf("Name => %s; want %s", got, want)
	}
}

func TestAttachCert(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	appName := "acme-inc"
	mustAppCreate(t, c, empire.App{
		Name: appName,
	})

	cert := "serverCertificate"
	app, err := c.AppUpdate(appName, &heroku.AppUpdateOpts{
		Cert: &cert,
	})
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, cert, app.Cert)
}

func TestAppList(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{
		Name: "acme-inc",
	})

	apps := mustAppList(t, c)

	if len(apps) != 1 {
		t.Fatal("Expected an app")
	}

	if got, want := apps[0].Name, "acme-inc"; got != want {
		t.Fatalf("Name => %s; want %s", got, want)
	}
}

func TestAppDelete(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{
		Name: "acme-inc",
	})

	mustAppDelete(t, c, "acme-inc")
}

func TestOrganizationAppCreate(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	app := mustOrganizationAppCreate(t, c, empire.App{
		Name: "acme-inc",
	})

	if got, want := app.Name, "acme-inc"; got != want {
		t.Fatalf("Name => %s; want %s", got, want)
	}
}

func TestAppDeploy(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	// App name should be different than acme-inc so we don't get a false
	// positive if the release is created based off the DefaultImage name
	appName := "my-app"

	mustAppCreate(t, c, empire.App{
		Name: appName,
	})

	mustAppDeploy(t, c, appName, DefaultImage)

	myAppReleases := mustReleaseList(t, c, appName)
	if len(myAppReleases) != 1 {
		t.Fatal("Expected a release")
	}

	if _, err := c.ReleaseList("acme-inc", nil); err == nil {
		t.Fatal("Expected no release for acme-inc")
	}

	// Deploy remind101/acme-inc which should infer acme-inc app name
	mustDeploy(t, c, DefaultImage)
	acmeIncAppReleases := mustReleaseList(t, c, "acme-inc")
	if len(acmeIncAppReleases) != 1 {
		t.Fatal("Expected a release for acme-inc")
	}
}

func TestAppDeployResourceDoesNotExist(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	var (
		f       DeployForm
		appName = "acme-inc"
	)

	f.Image = DefaultImage
	endpoint := fmt.Sprintf("/apps/%s/deploys", appName)

	if err := c.Post(ioutil.Discard, endpoint, &f); err == nil {
		t.Fatal("Expected resource not to exist")
	}
}

// Creates an app or fails.
func mustAppCreate(t testing.TB, c *heroku.Client, app empire.App) *heroku.App {
	name := string(app.Name)

	opts := heroku.AppCreateOpts{
		Name: &name,
	}

	a, err := c.AppCreate(&opts)
	if err != nil {
		t.Fatal(err)
	}

	return a
}

// Creates an app or fails.
func mustOrganizationAppCreate(t testing.TB, c *heroku.Client, app empire.App) *heroku.OrganizationApp {
	name := string(app.Name)

	opts := heroku.OrganizationAppCreateOpts{
		Name: &name,
	}

	a, err := c.OrganizationAppCreate(&opts, "")
	if err != nil {
		t.Fatal(err)
	}

	return a
}

// Lists apps or fails.
func mustAppList(t testing.TB, c *heroku.Client) []heroku.App {
	apps, err := c.AppList(nil)
	if err != nil {
		t.Fatal(err)
	}

	return apps
}

// Delets an app or fails.
func mustAppDelete(t testing.TB, c *heroku.Client, appName string) {
	if err := c.AppDelete(appName, ""); err != nil {
		t.Fatal(err)
	}
}

// Deploys an image for the app or fails.
func mustAppDeploy(t testing.TB, c *heroku.Client, appName string, image string) {
	var (
		f DeployForm
	)

	f.Image = image
	endpoint := fmt.Sprintf("/apps/%s/deploys", appName)

	if err := c.Post(ioutil.Discard, endpoint, &f); err != nil {
		t.Fatal(err)
	}
}
