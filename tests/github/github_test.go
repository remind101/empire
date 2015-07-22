package github_test

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"github.com/ejholmes/hookshot/events"
	"github.com/ejholmes/hookshot/hooker"
	"github.com/remind101/empire/empiretest"
)

func TestPing(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	if _, err := c.Ping(hooker.DefaultPing); err != nil {
		t.Fatal(err)
	}
}

func TestDeployment(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	var d events.Deployment
	d.Repository.FullName = "remind101/acme-inc"
	d.Deployment.ID = 1234
	d.Deployment.Ref = "master"
	d.Deployment.Sha = "827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57"
	d.Deployment.Task = "deployment"
	d.Deployment.Environment = "test"
	d.Deployment.Description = "Deploying"
	d.Deployment.Creator.Login = "ejholmes"

	resp, err := c.Trigger("deployment", &d)
	if err != nil {
		t.Fatal(err)
	}

	raw, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := resp.StatusCode, 202; got != want {
		t.Fatalf("StatusCode => %d; want %d", got, want)
	}

	if got, want := string(raw), "Ok\n"; got != want {
		t.Fatalf("Body => %q; want %q", got, want)
	}
}

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

// NewTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func NewTestClient(t testing.TB) (*hooker.Client, *httptest.Server) {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)

	c := hooker.NewClient(nil)
	c.URL = s.URL
	c.Secret = "abcd"

	return c, s
}
