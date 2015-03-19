package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

var (
	// DefaultURL is the default location for the GitHub API.
	DefaultURL = "https://api.github.com"
)

var (
	// errTwoFactor is returned when two factor authentication is required
	// to create an authorization for the user.
	errTwoFactor = errors.New("github: two factor required")

	// errUnauthorized is returned if the request to create an authorization
	// results in a 401.
	errUnauthorized = errors.New("github: unauthorized")
)

const (
	// HeaderTwoFactor is the HTTP header that github users for two factor
	// authentication.
	//
	// In a request, the value of this header should be the two factor auth
	// code. In a response, the presence of this header indicates that two
	// factor authentication is required for the user. See
	// http://goo.gl/h7al6K for more information.
	HeaderTwoFactor = "X-GitHub-OTP"
)

// CreateAuthorizationOpts is a set of options used when creating a GitHub
// authorization.
type CreateAuthorizationOpts struct {
	Scopes       []string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	TwoFactor    string
}

// Authorization represents a GitHub Authorization. See http://goo.gl/bs9I3o for
// more information.
type Authorization struct {
	Token string `json:"token"`
}

type User struct {
	Login string `json:"login"`
}

// Client is a github client.
type Client struct {
	// The github api url. The zero value is https://api.github.com.
	URL string

	client *http.Client
}

// CreateAuthorization creates a new GitHub authorization (or returns the
// existing authorization if present) for the GitHub OAuth application. See
// http://goo.gl/bs9I3o.
func (c *Client) CreateAuthorization(opts CreateAuthorizationOpts) (*Authorization, error) {
	f := struct {
		Scopes       []string `json:"scopes"`
		ClientSecret string   `json:"client_secret"`
	}{
		Scopes:       opts.Scopes,
		ClientSecret: opts.ClientSecret,
	}

	req, err := c.NewRequest("PUT", fmt.Sprintf("/authorizations/clients/%s", opts.ClientID), f)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(opts.Username, opts.Password)

	// If a two factor auth code is present, set the `X-GitHub-OTP` header
	// value. See http://goo.gl/Lumn6s.
	if opts.TwoFactor != "" {
		req.Header.Set(HeaderTwoFactor, opts.TwoFactor)
	}

	var a Authorization
	resp, err := c.Do(req, &a)
	if err != nil {
		return nil, err
	}

	// When the `X-GitHub-OTP` header is present in the response, it means
	// a two factor auth code needs to be provided.
	if resp.Header.Get(HeaderTwoFactor) != "" {
		return nil, errTwoFactor
	}

	if resp.StatusCode == 401 {
		return nil, errUnauthorized
	}

	if err := checkResponse(resp); err != nil {
		return nil, err
	}

	return &a, nil
}

// GetUser makes an authenticated request to /user and returns the User.
func (c *Client) GetUser(token string) (*User, error) {
	req, err := c.NewRequest("GET", "/user", nil)
	if err != nil {
		return nil, err
	}

	tokenAuth(req, token)

	var u User

	if _, err := c.Do(req, &u); err != nil {
		return nil, err
	}

	return &u, nil
}

// IsMember returns true of the authenticated user is a member of the
// organization.
func (c *Client) IsMember(organization, token string) (bool, error) {
	req, err := c.NewRequest("HEAD", fmt.Sprintf("/user/memberships/orgs/%s", organization), nil)
	if err != nil {
		return false, err
	}

	tokenAuth(req, token)

	resp, err := c.Do(req, nil)
	if err != nil {
		return false, err
	}

	if err := checkResponse(resp); err != nil {
		return false, nil
	}

	return true, nil
}

func (c *Client) NewRequest(method, path string, v interface{}) (*http.Request, error) {
	buf := new(bytes.Buffer)

	if err := json.NewEncoder(buf).Encode(v); err != nil {
		return nil, err
	}

	url := c.URL
	if url == "" {
		url = DefaultURL
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", url, path), buf)
	if err != nil {
		return req, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}

	if v != nil && resp.StatusCode/100 == 2 {
		defer resp.Body.Close()

		if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
			return resp, err
		}
	}

	return resp, nil
}

// tokenAuth sets the Authorization header in a request to use an OAuth token as
// the means of authentication. See http://goo.gl/kFTlnA.
func tokenAuth(req *http.Request, token string) {
	req.SetBasicAuth(token, "x-oauth-basic")
}

type errorResponse struct {
	Message string `json:"message"`
}

func (e *errorResponse) Error() string {
	return fmt.Sprintf("github: %s", e.Message)
}

func checkResponse(resp *http.Response) error {
	if resp.StatusCode/100 != 2 {
		var errResp errorResponse

		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return err
		}

		return &errResp
	}

	return nil
}
