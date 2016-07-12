package api_test

import (
	"testing"

	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/pkg/heroku"
)

var (

	// An test docker image that can be deployed.
	DefaultImage = "remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2"
)

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

// Client wraps an Empire test server and an hk client as a single unit.
type Client struct {
	*empiretest.Server
	*heroku.Client
}

// NewTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func NewTestClient(t testing.TB) *Client {
	e := empiretest.NewEmpire(t)
	s := empiretest.NewServer(t, e)

	c := &heroku.Client{
		Username: "",
		Password: "",
	}
	c.URL = s.URL

	return &Client{
		Client: c,
		Server: s,
	}
}

func (c *Client) Close() error {
	return c.Server.Close()
}
