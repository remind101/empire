package api_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
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

	a, err := c.OrganizationAppCreate(&opts)
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
	if err := c.AppDelete(appName); err != nil {
		t.Fatal(err)
	}
}
