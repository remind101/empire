package github

import (
	"io"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func newFakeClient(h http.Handler) (*Client, *httptest.Server) {
	s := httptest.NewServer(h)
	c := &Client{URL: s.URL}

	return c, s
}

func TestClientCreateAuthorization(t *testing.T) {
	c, s := newFakeClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()

		if !ok {
			t.Fatal("Expected credentials")
		}

		if got, want := user, "foo"; got != want {
			t.Fatalf("User => %s; want %s", got, want)
		}

		if got, want := pass, "bar"; got != want {
			t.Fatalf("Pass => %s; want %s", got, want)
		}

		if got, want := r.URL.Path, "/authorizations"; got != want {
			t.Fatalf("Path => %s; want %s", got, want)
		}

		body, err := ioutil.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := string(body), `{"scopes":["repo"],"client_id":"1234","client_secret":"shhh"}`+"\n"; got != want {
			t.Fatalf("Body => %s; want %s", got, want)
		}

		if len(r.Header[http.CanonicalHeaderKey(HeaderTwoFactor)]) != 0 {
			t.Fatalf("Two factor header should not have been set")
		}

		io.WriteString(w, `{"token":"abcd"}`)
	}))
	defer s.Close()

	a, err := c.CreateAuthorization(CreateAuthorizationOpts{
		Username:     "foo",
		Password:     "bar",
		Scopes:       []string{"repo"},
		ClientID:     "1234",
		ClientSecret: "shhh",
	})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := a.Token, "abcd"; got != want {
		t.Fatalf("Token => %s; want %s", got, want)
	}
}

func TestClientCreateAuthorizationTwoFactor(t *testing.T) {
	c, s := newFakeClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Header.Get("X-GitHub-OTP"), "code"; got != want {
			t.Fatalf("Two Factor Code => %s; want %s", got, want)
		}

		io.WriteString(w, `{"token":"token"}`)
	}))
	defer s.Close()

	_, err := c.CreateAuthorization(CreateAuthorizationOpts{
		TwoFactor: "code",
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestClientCreateAuthorizationTwoFactorRequired(t *testing.T) {
	c, s := newFakeClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-GitHub-OTP", "required; sms")
		w.WriteHeader(401)
		io.WriteString(w, `{}`)
	}))
	defer s.Close()

	_, err := c.CreateAuthorization(CreateAuthorizationOpts{
		TwoFactor: "code",
	})

	if err != errTwoFactor {
		t.Fatalf("err => %v; want %v", err, errTwoFactor)
	}
}

func TestClientCreateAuthorizationUnauthorized(t *testing.T) {
	c, s := newFakeClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer s.Close()

	_, err := c.CreateAuthorization(CreateAuthorizationOpts{})

	if err != errUnauthorized {
		t.Fatalf("err => %s; want %s", err, errUnauthorized)
	}
}

func TestClientCreateAuthorizationError(t *testing.T) {
	c, s := newFakeClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		io.WriteString(w, `{"message":"our SMS provider doesn't deliver to your area"}`)
	}))
	defer s.Close()

	_, err := c.CreateAuthorization(CreateAuthorizationOpts{})

	if got, want := err.Error(), "github: our SMS provider doesn't deliver to your area"; got != want {
		t.Fatalf("err => %s; want %s", got, want)
	}
}

func TestClientGetUser(t *testing.T) {
	c, s := newFakeClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		user, pass, ok := r.BasicAuth()
		if !ok {
			t.Fatal("Basic auth should be set")
		}

		if got, want := user, "token"; got != want {
			t.Fatalf("User => %s; want %s", got, want)
		}

		if got, want := pass, "x-oauth-basic"; got != want {
			t.Fatalf("Pass => %s; want %s", got, want)
		}

		io.WriteString(w, `{"login":"ejholmes"}`)
	}))
	defer s.Close()

	u, err := c.GetUser("token")
	if err != nil {
		t.Fatal(err)
	}

	if got, want := u.Login, "ejholmes"; got != want {
		t.Fatalf("Login => %s; want %s", got, want)
	}
}

func TestClientIsMember(t *testing.T) {
	tests := []struct {
		status int
		member bool
	}{
		{200, true},
		{204, true},
		{404, false},
	}

	for _, tt := range tests {
		c, s := newFakeClient(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(tt.status)
		}))
		defer s.Close()

		ok, err := c.IsMember("remind101", "token")
		if err != nil {
			t.Fatal(err)
		}

		if got, want := ok, tt.member; got != want {
			t.Fatalf("IsMember for %d status => %v; want %v", tt.status, got, want)
		}
	}
}
