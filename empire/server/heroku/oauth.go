package heroku

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/remind101/empire/empire"
	"golang.org/x/net/context"
)

const (
	HeaderTwoFactor       = "Heroku-Two-Factor-Code"
	HeaderGitHubTwoFactor = "X-GitHub-OTP"
)

// DefaultGitHubScopes is the default oauth scopes to obtain when getting an
// authorization from GitHub.
var DefaultGitHubScopes = []string{
	"repo_deployment", // For creating deployment statuses.
	"read:org",        // For reading organization memberships.
}

// Authorization represents a response to create an access token.
type Authorization struct {
	AccessToken empire.AccessToken `json:"access_token"`
}

type PostAuthorizations struct {
	*empire.Empire
	Authorizer
}

func (h *PostAuthorizations) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	user, pass, ok := r.BasicAuth()
	if !ok {
		return ErrBadRequest
	}

	u, err := h.Authorize(user, pass, r.Header.Get(HeaderTwoFactor))
	if err != nil {
		return err
	}

	at, err := h.Empire.AccessTokensCreate(&empire.AccessToken{
		User: u,
	})
	if err != nil {
		return err
	}

	auth := Authorization{AccessToken: *at}

	return Encode(w, auth)
}

// Authorizer is an interface for obtaining an authorization.
type Authorizer interface {
	Authorize(username, password, twofactor string) (*empire.User, error)
}

// NewAuthorizer returns a new Authorizer. If the client id is present, it will
// return a real Authorizer that talks to GitHub. If an empty string is
// provided, then it will just return a fake authorizer.
func NewAuthorizer(clientID, clientSecret, organization string) Authorizer {
	if clientID == "" {
		return &authorizer{}
	}

	return &GitHubAuthorizer{
		Scopes:       DefaultGitHubScopes,
		ClientID:     clientID,
		ClientSecret: clientSecret,
		Organization: organization,
	}
}

// GitHubAuthorizer is an implementation of the Authorizer interface backed by
// GitHub's Non-Web Application Flow, which can be found at
// http://goo.gl/onpQKM.
type GitHubAuthorizer struct {
	// OAuth scopes that should be granted with the access token.
	Scopes []string

	// The oauth application client id.
	ClientID string

	// The oauth application client secret.
	ClientSecret string

	// If provided, it will ensure that the user is a member of this
	// organization.
	Organization string

	url string
}

func (a *GitHubAuthorizer) Authorize(username, password, twofactor string) (*empire.User, error) {
	f := struct {
		Scopes       []string `json:"scopes"`
		ClientSecret string   `json:"client_secret"`
	}{
		Scopes:       a.Scopes,
		ClientSecret: a.ClientSecret,
	}

	raw, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	if a.url == "" {
		a.url = "https://api.github.com"
	}

	req, err := http.NewRequest("PUT", a.url+"/authorizations/clients/"+a.ClientID, bytes.NewReader(raw))
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.SetBasicAuth(username, password)

	// If a two factor auth code is present, set the `X-GitHub-OTP` header
	// value. See http://goo.gl/Lumn6s.
	if twofactor != "" {
		req.Header.Set(HeaderGitHubTwoFactor, twofactor)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, ErrTwoFactor
	}

	// When the `X-GitHub-OTP` header is present in the response, it means
	// a two factor auth code needs to be provided.
	if resp.Header.Get(HeaderGitHubTwoFactor) != "" {
		return nil, ErrTwoFactor
	}

	var ga struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&ga); err != nil {
		return nil, err
	}

	token := ga.Token

	req, err = http.NewRequest("GET", a.url+"/user", nil)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.SetBasicAuth(token, "x-oauth-basic")

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode/100 != 2 {
		return nil, ErrUnauthorized
	}

	var gu struct {
		Login string `json:"login"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&gu); err != nil {
		return nil, err
	}

	user := &empire.User{
		Name:        gu.Login,
		GitHubToken: token,
	}

	if a.Organization != "" {
		req, err = http.NewRequest("HEAD", a.url+"/user/memberships/orgs/"+a.Organization, nil)
		if err != nil {
			return user, err
		}
		req.Header.Set("Accept", "application/vnd.github.v3+json")
		req.SetBasicAuth(token, "x-oauth-basic")

		resp, err = http.DefaultClient.Do(req)
		if err != nil {
			return user, err
		}

		if resp.StatusCode/100 != 2 {
			return user, &ErrorResource{
				Status:  http.StatusForbidden,
				ID:      "forbidden",
				Message: fmt.Sprintf("You are not a member of %s", a.Organization),
			}
		}
	}

	return user, nil
}

// authorizer is a fake implementation of the Authorizer interface that let's
// anyone in. Used in development.
type authorizer struct{}

// Authorizer implements Authorizer Authorize.
func (a *authorizer) Authorize(username, password, twofactor string) (*empire.User, error) {
	return &empire.User{Name: "fake", GitHubToken: "token"}, nil
}
