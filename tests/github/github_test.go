package github_test

import (
	"io/ioutil"
	"testing"

	"golang.org/x/net/context"

	"github.com/ejholmes/hookshot/events"
	"github.com/ejholmes/hookshot/hooker"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestPing(t *testing.T) {
	c := newClient(t)
	defer c.Close()

	if _, err := c.Ping(hooker.DefaultPing); err != nil {
		t.Fatal(err)
	}
}

func TestDeployment(t *testing.T) {
	c := newClient(t)
	defer c.Close()

	s := new(mockScheduler)
	s.image = make(chan string, 1)
	c.Empire.Scheduler = s

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

type client struct {
	*empiretest.Server
	*hooker.Client
}

// newClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func newClient(t testing.TB) *client {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)

	c := hooker.NewClient(nil)
	c.URL = s.URL()
	c.Secret = "abcd"

	return &client{
		Server: s,
		Client: c,
	}
}

type mockScheduler struct {
	scheduler.Scheduler
	image chan string
}

func (m *mockScheduler) Submit(_ context.Context, app *scheduler.App, ss scheduler.StatusStream) error {
	m.image <- app.Processes[0].Image.String()
	return nil
}
