package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGitHubAuthorizer(t *testing.T) {
	tests := []struct {
		handler   http.Handler // http.Handler for PUT api.github.com/authorizations/client/:id
		twofactor string

		token string
		err   error
	}{
		{
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/authorizations/clients/":
					user, pass, ok := r.BasicAuth()

					if !ok {
						t.Fatal("Expected basic auth to be set")
					}

					if got, want := user, "user"; got != want {
						t.Fatalf("User => %q; want %q", got, want)
					}

					if got, want := pass, "pass"; got != want {
						t.Fatalf("Password => %q; want %q", got, want)
					}

					if len(r.Header["X-Github-Otp"]) > 0 {
						t.Fatal("Expected X-GitHub-OTP to not be set")
					}

					io.WriteString(w, `{"token":"token"}`)
				case "/user":
					io.WriteString(w, `{"login":"foobar"}`)
				}
			}),
			twofactor: "",
			token:     "token",
			err:       nil,
		},
		{
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/authorizations/clients/":
					if got, want := r.Header.Get("X-Github-Otp"), "abc"; got != want {
						t.Fatalf("X-GitHub-OTP Header => %q; want %q", got, want)
					}

					io.WriteString(w, `{"token":"token"}`)
				case "/user":
					io.WriteString(w, `{"login":"foobar"}`)
				}
			}),
			twofactor: "abc",
			token:     "token",
			err:       nil,
		},
		{
			handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(401)
				io.WriteString(w, `{}`)
			}),
			twofactor: "",
			token:     "",
			err:       ErrTwoFactor,
		},
	}

	for _, tt := range tests {
		s := httptest.NewServer(tt.handler)
		defer s.Close()

		auth := &GitHubAuthorizer{url: s.URL}

		user, err := auth.Authorize("user", "pass", tt.twofactor)
		if err != tt.err {
			t.Fatalf("Error => %v; want %v", err, tt.err)
			continue
		}

		if user != nil {
			if got, want := user.GitHubToken, tt.token; got != want {
				t.Fatalf("Token => %q; want %q", got, want)
			}
		}
	}
}
