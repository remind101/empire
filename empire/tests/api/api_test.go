package api_test

import (
	"net/http/httptest"
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empiretest"
)

var (

	// An test docker image that can be deployed.
	DefaultImage = empire.Image{
		Repo: "quay.io/ejholmes/acme-inc",
		ID:   "ec238137726b58285f8951802aed0184f915323668487b4919aff2671c0f9a02",
	}
)

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

// NewTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func NewTestClient(t testing.TB) (*heroku.Client, *httptest.Server) {
	s := empiretest.NewServer(t)
	c := &heroku.Client{}
	c.URL = s.URL

	return c, s
}
