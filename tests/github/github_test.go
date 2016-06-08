package github_test

import (
	"io/ioutil"
	"net/http/httptest"
	"testing"

	"golang.org/x/net/context"

	"github.com/ejholmes/hookshot/events"
	"github.com/ejholmes/hookshot/hooker"
	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	e := empiretest.NewEmpire(t)
	c, s := NewTestClient(t, e)
	defer s.Close()

	if _, err := c.Ping(hooker.DefaultPing); err != nil {
		t.Fatal(err)
	}
}

func TestDeployment(t *testing.T) {
	e := empiretest.NewEmpire(t)
	s := new(mockScheduler)
	s.image = make(chan string, 1)
	e.Scheduler = s

	c, sv := NewTestClient(t, e)
	defer sv.Close()

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
	assert.NoError(t, err)

	raw, err := ioutil.ReadAll(resp.Body)
	assert.NoError(t, err)
	assert.Equal(t, 202, resp.StatusCode)
	assert.Equal(t, "Ok\n", string(raw))
	assert.Equal(t, "remind101/acme-inc:827fecd2d36ebeaa2fd05aa8ef3eed1e56a8cd57", <-s.image)
}

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

// NewTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func NewTestClient(t testing.TB, e *empire.Empire) (*hooker.Client, *httptest.Server) {
	s := empiretest.NewServer(t, e)

	c := hooker.NewClient(nil)
	c.URL = s.URL
	c.Secret = "abcd"

	return c, s
}

type mockScheduler struct {
	scheduler.Scheduler
	image chan string
}

func (m *mockScheduler) Submit(_ context.Context, app *scheduler.App) error {
	m.image <- app.Processes[0].Image
	return nil
}
