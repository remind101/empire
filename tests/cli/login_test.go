package cli_test

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"
	"testing"

	"github.com/remind101/empire"
	"github.com/remind101/empire/empiretest"
	"github.com/remind101/empire/server/auth"
	"github.com/remind101/empire/server/heroku"
)

func TestLogin(t *testing.T) {
	cli := newCLI(t)
	defer cli.Close()
	cli.Server.Heroku.Auth = newAuth(auth.StaticAuthenticator("fake", "bar", "", &empire.User{Name: "fake"}))
	cli.Start()

	input := "fake\nbar\n"

	cmd := cli.Command("login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(out), "The login command is deprecated and will stop working in Nov 2020.  Please use weblogin.\nEnter email: Logged in.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestLoginUnauthorized(t *testing.T) {
	cli := newCLI(t)
	defer cli.Close()
	cli.Server.Heroku.Auth = newAuth(auth.StaticAuthenticator("fake", "bar", "", &empire.User{Name: "fake"}))
	cli.Start()

	input := "foo\nbar\n"

	cmd := cli.Command("login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatal("Expected an error")
	}

	if got, want := string(out), "The login command is deprecated and will stop working in Nov 2020.  Please use weblogin.\nEnter email: error: Request not authenticated, API token is missing, invalid or expired Log in with `emp login`.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestLoginTwoFactor(t *testing.T) {
	cli := newCLI(t)
	defer cli.Close()
	cli.Server.Heroku.Auth = newAuth(auth.StaticAuthenticator("twofactor", "bar", "code", &empire.User{Name: "fake"}))
	cli.Start()

	input := "twofactor\nbar\ncode\n"

	cmd := cli.Command("login")
	cmd.Stdin = strings.NewReader(input)

	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatal(err)
	}

	if got, want := string(out), "The login command is deprecated and will stop working in Nov 2020.  Please use weblogin.\nEnter email: Enter two-factor auth code: Logged in.\n"; got != want {
		t.Fatalf("%q", got)
	}
}

func TestLoginSAML(t *testing.T) {
	cli := newCLI(t)
	defer cli.Close()

	loginURL := fmt.Sprintf("%s/saml/login", cli.Server.URL())
	cli.Server.Heroku.Unauthorized = heroku.SAMLUnauthorized(loginURL)

	idp := empiretest.NewIdentityProvider()
	defer idp.Close()
	cli.Server.ServiceProvider = idp.AddServiceProvider(cli.Server.URL())

	cli.Start()

	cli.RunCommands(t, []Command{
		{
			"apps",
			fmt.Errorf("error: Request not authenticated, API token is missing, invalid or expired. Login at %s", loginURL),
		},
	})

	// Get an API token via a SAML service provider initiated login. This
	// simulates the user clicking the link returned above.
	token, err := serviceProviderLogin(loginURL)
	if err != nil {
		t.Fatal(err)
	}

	if err := cli.Authorize("dummy", token); err != nil {
		t.Fatal(err)
	}

	// CLI should not be authenticated.
	cli.RunCommands(t, []Command{
		{
			"apps",
			"",
		},
	})
}

// serviceProviderLogin starts a Service Provider initiated SAML login,
// returning an API token.
func serviceProviderLogin(loginURL string) (string, error) {
	cookies, _ := cookiejar.New(nil)
	client := &http.Client{
		Jar: cookies,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// We expect two redirects to get the SAMLResponse. We
			// don't want to execute the last redirect, because
			// we'll extract information from the Location and use
			// an HTTP POST instead.
			if len(via) >= 2 {
				return http.ErrUseLastResponse
			}
			return nil
		},
	}

	// Let's start with a Service Provider initiated login. We'll hit the
	// /saml/login endpoint in Empire, which will:
	//
	//	1. Generate a SAML AuthnRequest.
	//	2. Redirect the to the /sso endpoint on the IdP to
	//	"authenticate" the user.
	req, _ := http.NewRequest("GET", loginURL, nil)
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	io.Copy(ioutil.Discard, resp.Body)

	if resp.StatusCode != 307 {
		return "", fmt.Errorf("expected a redirect back to the SAML Assertion Consumer Service URL")
	}

	// At this point in time, the IdP has given us back the RelayState and
	// SAMLResponse values in a redirect. We'll extract these values and
	// POST them to the given URL.
	//
	// In a real IdP, this would be automatically posted via an HTML form.
	acsURL, err := url.Parse(resp.Header.Get("Location"))
	if err != nil {
		return "", err
	}

	form := acsURL.Query()
	acsURL.RawQuery = ""
	req, _ = http.NewRequest("POST", acsURL.String(), strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	// For testing purposes, request text/plain which will give us back the
	// raw API token instead of html.
	req.Header.Set("Accept", "text/plain")
	resp, err = client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	// Decode the API token from the body, and write it to ~/.netrc.
	var buf bytes.Buffer
	io.Copy(&buf, resp.Body)
	return buf.String(), nil
}
