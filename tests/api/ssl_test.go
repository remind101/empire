package api_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
)

func TestSSLEndpoint(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{Name: "acme-inc"})
	mustSSLEndpointsCreate(t, c, "acme-inc", "CertificateChain", "PrivateKey")

	endpoints := mustSSLEndpointsList(t, c, "acme-inc")

	if len(endpoints) != 1 {
		t.Fatal("Expected an SSL endpoint")
	}

	if got, want := endpoints[0].Name, "fake"; got != want {
		t.Fatalf("Name => %s; want %s", got, want)
	}

	mustSSLEndpointsDelete(t, c, "acme-inc", endpoints[0].Id)

	endpoints = mustSSLEndpointsList(t, c, "acme-inc")
	if len(endpoints) != 0 {
		t.Fatal("Expected no SSL endpoints")
	}
}

func mustSSLEndpointsCreate(t *testing.T, c *heroku.Client, app string, cert string, key string) *heroku.SSLEndpoint {
	e, err := c.SSLEndpointCreate(app, cert, key, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func mustSSLEndpointsList(t *testing.T, c *heroku.Client, app string) []heroku.SSLEndpoint {
	e, err := c.SSLEndpointList(app, nil)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func mustSSLEndpointsDelete(t *testing.T, c *heroku.Client, app string, cert string) {
	if err := c.SSLEndpointDelete(app, cert); err != nil {
		t.Fatal(err)
	}
}
