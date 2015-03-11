package github_test

import (
	"net/http/httptest"

	"testing"

	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/pkg/hooker"
	"github.com/remind101/empire/empire/server/github"
	"github.com/remind101/empire/empiretest"
)

var (
	// DefaultCommit is a commit the commit that corresponds to the
	// DefaultImage.
	DefaultCommit = empire.Commit{
		Repo: "ejholmes/acme-inc",
		Sha:  "66a675359fc5077881bbd57cef20429e43481667",
	}
)

func TestDeployment(t *testing.T) {
	c, _, s := NewTestClient(t)
	defer s.Close()

	var p github.Deployment
	p.Repository.FullName = "ejholmes/acme-inc"
	p.Deployment.Sha = "66a675359fc5077881bbd57cef20429e43481667"

	_, err := c.Trigger("deployment", &p)
	if err != nil {
		t.Fatal(err)
	}
}

// This tests the scenario where an app has already been created, but does not
// have a linked github/docker repo.
func TestDeploymentAppExists(t *testing.T) {
	c, e, s := NewTestClient(t)
	defer s.Close()

	_, err := e.AppsCreate(&empire.App{
		Name: "acme-inc",
	})
	if err != nil {
		t.Fatal(err)
	}

	var p github.Deployment
	p.Repository.FullName = "ejholmes/acme-inc"
	p.Deployment.Sha = "66a675359fc5077881bbd57cef20429e43481667"

	_, err = c.Trigger("deployment", &p)
	if err != nil {
		t.Fatal(err)
	}

	app, err := e.AppsFind("acme-inc")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := *app.Repos.Docker, empire.Repo("quay.io/ejholmes/acme-inc"); got != want {
		t.Fatalf("App.Repos.Docker => %s; want %s", got, want)
	}
}

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

// NewTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func NewTestClient(t testing.TB) (*hooker.Client, *empire.Empire, *httptest.Server) {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)
	c := hooker.NewClient(nil)
	c.URL = s.URL

	return c, e, s
}
