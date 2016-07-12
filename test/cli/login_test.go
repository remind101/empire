package cli_test

import (
	"strings"
	"testing"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/test"
)

func TestLogin(t *testing.T) {
	s := test.NewTestServer(t, nil, server.Options{
		Authenticator: auth.StaticAuthenticator("fake", "bar", "", &empire.User{Name: "fake"}),
	})

	cli := newCLIWithServer(t, s)
	defer cli.Close()

	input := "fake\nbar\n"

	cmd := cli.Exec("login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(out), "Enter email: Logged in.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	s := test.NewTestServer(t, nil, server.Options{
		Authenticator: auth.StaticAuthenticator("fake", "bar", "", &empire.User{Name: "fake"}),
	})

	cli := newCLIWithServer(t, s)
	defer cli.Close()

	input := "foo\nbar\n"

	cmd := cli.Exec("login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected an error")
	}

	if got, want := string(out), "Enter email: error: Request not authenticated, API token is missing, invalid or expired Log in with `emp login`.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestLoginTwoFactor(t *testing.T) {
	s := test.NewTestServer(t, nil, server.Options{
		Authenticator: auth.StaticAuthenticator("twofactor", "bar", "code", &empire.User{Name: "fake"}),
	})

	cli := newCLIWithServer(t, s)
	defer cli.Close()

	input := "twofactor\nbar\ncode\n"

	cmd := cli.Exec("login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(out), "Enter email: Enter two-factor auth code: Logged in.\n"; got != want {
		t.Fatalf("%q", got)
	}
}
