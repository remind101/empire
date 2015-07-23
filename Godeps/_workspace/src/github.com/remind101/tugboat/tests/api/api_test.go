package api_test

import (
	"fmt"
	"io/ioutil"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/remind101/tugboat"
	"github.com/remind101/tugboat/tugboattest"
)

// Run the tests with tugboattest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	tugboattest.Run(m)
}

func TestDeploymentsCreate(t *testing.T) {
	c, _, s := newTestClient(t)
	defer s.Close()

	d, err := createDeployment(c)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := d.Repo, "remind101/acme-inc"; got != want {
		t.Fatalf("Repo => %s; want %s", got, want)
	}

	if got, want := d.Status, tugboat.StatusStarted; got != want {
		t.Fatalf("Status => %s; want %s", got, want)
	}

	if got, want := d.Environment, "production"; got != want {
		t.Fatalf("Environment => %s; want %s", got, want)
	}

	if got, want := d.Provider, "heroku"; got != want {
		t.Fatalf("Provider => %s; want %s", got, want)
	}
}

func TestDeploymentsCreate_Unauthorized(t *testing.T) {
	c, _, s := newTestClient(t)
	defer s.Close()

	u, err := url.Parse(c.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.User("")
	c.URL = u.String()

	if _, err := createDeployment(c); err == nil {
		t.Fatal("Expected request to not be authorized")
	}
}

func TestStreamLogs(t *testing.T) {
	c, tug, s := newTestClient(t)
	defer s.Close()

	d, err := createDeployment(c)
	if err != nil {
		t.Fatal(err)
	}

	logs := `Logs
Are
Awesome!`

	if err := c.WriteLogs(d, strings.NewReader(logs)); err != nil {
		t.Fatal(err)
	}

	got, err := tug.Logs(d)
	if err != nil {
		t.Fatal(err)
	}

	if want := logs; got != want {
		t.Fatalf("Logs => %q; want %q", got, want)
	}
}

func TestUpdateStatus(t *testing.T) {
	c, _, s := newTestClient(t)
	defer s.Close()

	d, err := createDeployment(c)
	if err != nil {
		t.Fatal(err)
	}

	if err := c.UpdateStatus(d, tugboat.StatusUpdate{
		Status: tugboat.StatusSucceeded,
	}); err != nil {
		t.Fatal(err)
	}
}

// createDeployment creates a fake Deployment.
func createDeployment(c *tugboat.Client) (*tugboat.Deployment, error) {
	return c.DeploymentsCreate(tugboat.DeployOpts{
		ID:          354773,
		Sha:         "f6044cf59b8dc26af97e1ebd9b955c39d7baeb74",
		Ref:         "master",
		Environment: "production",
		Description: "Deployment",
		Repo:        "remind101/acme-inc",
		User:        "ejholmes",
		Provider:    "heroku",
	})
}

// newTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func newTestClient(t testing.TB) (*tugboat.Client, *tugboat.Tugboat, *httptest.Server) {
	tug := tugboattest.New(t)

	s := httptest.NewServer(tugboattest.NewServer(tug))
	c := tugboat.NewClient(nil)
	u, err := url.Parse(s.URL)
	if err != nil {
		t.Fatal(err)
	}
	u.User = url.User("eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJQcm92aWRlciI6Imhlcm9rdSJ9.HVBoIvRnGKR87odScLnkFWHi4pvSI8V7LJpjh00njBY")
	c.URL = u.String()

	return c, tug, s
}

func deploymentPayload(t testing.TB, fixture string) []byte {
	raw, err := ioutil.ReadFile(fmt.Sprintf("test-fixtures/%s.json", fixture))
	if err != nil {
		t.Fatal(err)
	}

	return raw
}
