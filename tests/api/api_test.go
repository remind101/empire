package api_test

import (
	"net/http/httptest"
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
)

var (
	// DatabaseURL is a connection string for the postgres database to use
	// during integration tests.
	DatabaseURL = "postgres://localhost/empire?sslmode=disable"

	// An test docker image that can be deployed.
	DefaultImage = empire.Image{
		Repo: "ejholmes/acme-inc",
		ID:   "ec238137726b58285f8951802aed0184f915323668487b4919aff2671c0f9a02",
	}
)

// NewTestClient will return a new heroku.Client that's configured to interact
// with a instance of the empire HTTP server.
func NewTestClient(t testing.TB) (*heroku.Client, *httptest.Server) {
	opts := empire.Options{DB: DatabaseURL}

	e, err := empire.New(opts)
	if err != nil {
		t.Fatal(err)
	}

	if err := e.Reset(); err != nil {
		t.Fatal(err)
	}

	s := httptest.NewServer(empire.NewServer(e))
	c := &heroku.Client{}
	c.URL = s.URL

	return c, s
}
