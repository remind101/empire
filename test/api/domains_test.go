package api_test

import (
	"testing"

	"github.com/remind101/empire"
)

func TestDomainCreate(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	mustAppCreate(t, c.Client, empire.App{Name: "acme-inc"})

	d, err := c.DomainCreate("acme-inc", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := d.Hostname, "example.com"; got != want {
		t.Fatalf("Hostname => %s; want %s", got, want)
	}

}

func TestDomainCreateAlreadyInUse(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	mustAppCreate(t, c.Client, empire.App{Name: "acme-inc"})
	mustAppCreate(t, c.Client, empire.App{Name: "acme-corp"})

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
	c := NewTestClient(t)
	defer c.Close()

	mustAppCreate(t, c.Client, empire.App{Name: "acme-inc"})

	_, err := c.DomainCreate("acme-inc", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.DomainCreate("acme-inc", "example.com")
	if got, want := err.Error(), "example.com is already added to this app."; got != want {
		t.Fatalf("DomainCreate() => %s; want %s", got, want)
	}
}

func TestDomainDestroy(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	mustAppCreate(t, c.Client, empire.App{Name: "acme-inc"})

	_, err := c.DomainCreate("acme-inc", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	if err := c.DomainDelete("acme-inc", "example.com"); err != nil {
		t.Fatal(err)
	}
}

func TestDomainDestroyNotFound(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	mustAppCreate(t, c.Client, empire.App{Name: "acme-inc"})
	mustAppCreate(t, c.Client, empire.App{Name: "acme-corp"})

	// Try to remove non existent domain
	err := c.DomainDelete("acme-inc", "example.com")
	if got, want := err.Error(), "Couldn't find that domain name."; got != want {
		t.Fatalf("DomainDelete() => %s; want %s", got, want)
	}

	// Add domain to acme-corp
	_, err = c.DomainCreate("acme-corp", "example.com")
	if err != nil {
		t.Fatal(err)
	}

	// Try to remove from the wrong app
	err = c.DomainDelete("acme-inc", "example.com")
	if got, want := err.Error(), "Couldn't find that domain name."; got != want {
		t.Fatalf("DomainDelete() => %s; want %s", got, want)
	}
}
