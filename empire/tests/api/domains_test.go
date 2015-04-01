package api_test

import (
	"testing"

	"github.com/remind101/empire/empire"
)

func TestDomainCreate(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{Name: "acme-inc"})

	d, err := c.DomainCreate("acme-inc", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := d.Hostname, "example.com"; got != want {
		t.Fatalf("Hostname => %s; want %s", got, want)
	}

}

func TestDomainCreateAlreadyInUse(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{Name: "acme-inc"})
	mustAppCreate(t, c, empire.App{Name: "acme-corp"})

	_, err := c.DomainCreate("acme-inc", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.DomainCreate("acme-corp", "example.com")
	if got, want := err.Error(), "example.com is currently in use by another app."; got != want {
		t.Fatalf("DomainCreate() => %s; want %s", got, want)
	}
}

func TestDomainCreateAlreadyAdded(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustAppCreate(t, c, empire.App{Name: "acme-inc"})

	_, err := c.DomainCreate("acme-inc", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.DomainCreate("acme-inc", "example.com")
	if got, want := err.Error(), "example.com is already added to this app."; got != want {
		t.Fatalf("DomainCreate() => %s; want %s", got, want)
	}
}
