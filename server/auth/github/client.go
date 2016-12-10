package github

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/remind101/empire/tracer"

	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

var (
	// DefaultURL is the default location for the GitHub API.
	DefaultURL = "https://api.github.com"

	// The number of times that GET requests will be retried in the event of
	// an error.
	DefaultGetRetries = 2
)

var (
	// errTwoFactor is returned when two factor authentication is required
	// to create an authorization for the user.
	errTwoFactor = errors.New("github: two factor required")

	// errUnauthorized is returned if the request to create an authorization
	// results in a 401.
	errUnauthorized = errors.New("github: unauthorized")

	// errNoToken is returned if there was no access token in the github
	// response.
	errNoToken = errors.New("github: no token in response")
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

// CreateAuthorizationOptions is a set of options used when creating a GitHub
// authorization.
type CreateAuthorizationOptions struct {
	Username string
	Password string
	OTP      string
}

// Client is a github client implementation for creating authorizations, and
// checking organization membership.
type Client struct {
	// URL for the GitHub API. This can be changed if you're using GitHub
	// Enterprise. The zero value is DefaultURL.
	URL string

	// OAuth configuration.
	*oauth2.Config

	client interface {
		Do(*http.Request) (*http.Response, error)
	}

	// should return the amount of time to wait until the next try.
	backoff func(try int) time.Duration
}

func backoff(try int) time.Duration {
	return time.Duration(try+1) * (time.Second * 1)
}

// NewClient returns a new Client instance that uses the given oauth2 config.
func NewClient(config *oauth2.Config) *Client {
	return &Client{
		Config:  config,
		backoff: backoff,
	}
}

// Authorization represents a GitHub Authorization. See http://goo.gl/bs9I3o for
// more information.
type Authorization struct {
	Token string `json:"token"`
}

type User struct {
	Login string `json:"login"`
}

type TeamMembership struct {
	State string `json:"state"`
}

// CreateAuthorization creates a new GitHub authorization (or returns the
// existing authorization if present) for the GitHub OAuth application. See
// http://goo.gl/bs9I3o.
func (c *Client) CreateAuthorization(ctx context.Context, opts CreateAuthorizationOptions) (*Authorization, error) {
	span := newSpan(ctx, "CreateAuthorization")
	span.SetMeta("username", opts.Username)
	span.SetMeta("client_id", c.ClientID)
	span.SetMeta("scopes", strings.Join(c.Scopes, ","))
	authorization, err := c.createAuthorization(span.Context(ctx), opts)
	span.FinishWithErr(err)
	return authorization, err
}

func (c *Client) createAuthorization(ctx context.Context, opts CreateAuthorizationOptions) (*Authorization, error) {
	f := struct {
		Scopes       []string `json:"scopes"`
		ClientID     string   `json:"client_id"`
		ClientSecret string   `json:"client_secret"`
	}{
		Scopes:       c.Scopes,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
	}

	req, err := c.NewRequest("POST", "/authorizations", f)
	if err != nil {
		return nil, err
	}

	req.SetBasicAuth(opts.Username, opts.Password)

	// If a two factor auth code is present, set the `X-GitHub-OTP` header
	// value. See http://goo.gl/Lumn6s.
	if opts.OTP != "" {
		req.Header.Set(HeaderTwoFactor, opts.OTP)
	}

	var a Authorization
	resp, err := c.Do(req, &a)
	if err != nil {
		if resp != nil && resp.StatusCode == 401 {
			// When the `X-GitHub-OTP` header is present in the response, it means
			// a two factor auth code needs to be provided.
			if resp.Header.Get(HeaderTwoFactor) != "" {
				return nil, errTwoFactor
			}

			return nil, errUnauthorized
		}

		return nil, err
	}

	if a.Token == "" {
		return nil, errNoToken
	}

	return &a, nil
}

// GetUser makes an authenticated request to /user and returns the GitHub User.
func (c *Client) GetUser(ctx context.Context, token string) (*User, error) {
	span := newSpan(ctx, "GetUser")
	if token != "" {
		span.SetMeta("token", "XXXX")
	}
	user, err := c.getUser(span.Context(ctx), token)
	span.FinishWithErr(err)
	return user, err
}

func (c *Client) getUser(ctx context.Context, token string) (*User, error) {
	req, err := c.NewRequest("GET", "/user", nil)
	if err != nil {
		return nil, err
	}

	tokenAuth(req, token)

	var u User

	_, err = c.Do(req, &u)
	if err != nil {
		return nil, err
	}

	return &u, nil
}

// IsOrganizationMember returns true of the authenticated user is a member of the
// organization.
func (c *Client) IsOrganizationMember(ctx context.Context, organization, token string) (bool, error) {
	span := newSpan(ctx, "IsOrganizationMember")
	span.SetMeta("organization", organization)
	if token != "" {
		span.SetMeta("token", "XXXX")
	}
	ok, err := c.isOrganizationMember(span.Context(ctx), organization, token)
	span.FinishWithErr(err)
	return ok, err
}

func (c *Client) isOrganizationMember(ctx context.Context, organization, token string) (bool, error) {
	req, err := c.NewRequest("HEAD", fmt.Sprintf("/user/memberships/orgs/%s", organization), nil)
	if err != nil {
		return false, err
	}

	tokenAuth(req, token)

	resp, err := c.Do(req, nil)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}

	return true, nil
}

// IsTeamMember returns true if the given user is a member of the team.
func (c *Client) IsTeamMember(ctx context.Context, teamID, token string) (bool, error) {
	span := newSpan(ctx, "IsTeamMember")
	span.SetMeta("team_id", teamID)
	if token != "" {
		span.SetMeta("token", "XXXX")
	}
	ok, err := c.isTeamMember(span.Context(ctx), teamID, token)
	span.FinishWithErr(err)
	return ok, err
}

func (c *Client) isTeamMember(ctx context.Context, teamID, token string) (bool, error) {
	u, err := c.GetUser(ctx, token)
	if err != nil {
		return false, err
	}

	req, err := c.NewRequest("GET", fmt.Sprintf("/teams/%s/memberships/%s", teamID, u.Login), nil)
	if err != nil {
		return false, err
	}

	tokenAuth(req, token)

	var t TeamMembership

	resp, err := c.Do(req, &t)
	if err != nil {
		if resp != nil && resp.StatusCode == 404 {
			return false, nil
		}
		return false, err
	}

	return t.State == "active", nil
}

func (c *Client) NewRequest(method, path string, v interface{}) (*http.Request, error) {
	var r io.Reader
	if v != nil {
		buf := new(bytes.Buffer)

		if err := json.NewEncoder(buf).Encode(v); err != nil {
			return nil, err
		}

		r = buf
	}

	url := c.URL
	if url == "" {
		url = DefaultURL
	}

	req, err := http.NewRequest(method, fmt.Sprintf("%s%s", url, path), r)
	if err != nil {
		return req, err
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")

	return req, nil
}

func (c *Client) Do(req *http.Request, v interface{}) (*http.Response, error) {
	return c.do(req, v, 0)
}

func (c *Client) do(req *http.Request, v interface{}, try int) (*http.Response, error) {
	client := c.client
	if client == nil {
		client = http.DefaultClient
	}

	resp, err := client.Do(req)
	if err != nil {
		return resp, err
	}

	if err := checkResponse(resp); err != nil {
		if requestRetryable(req, resp) && try < DefaultGetRetries {
			time.Sleep(c.backoff(try))
			return c.do(req, v, try+1)
		}
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
		defer resp.Body.Close()
		var errResp errorResponse

		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return err
		}

		return &errResp
	}

	return nil
}

// Returns true if the request is retryable. Only idempotent requests that
// return a 401 are retried.
//
// This is done to address an issue in the GitHub API, where newly create auth
// tokens aren't immediately available, presumably because GitHub uses a read
// replica.
//
// See https://github.com/remind101/empire/issues/1026
func requestRetryable(req *http.Request, resp *http.Response) bool {
	idempotent := req.Method == "GET" || req.Method == "HEAD"
	return idempotent && resp.StatusCode == 401
}

func newSpan(ctx context.Context, name string) *tracer.Span {
	span := tracer.NewChildSpanFromContext(name, ctx)
	span.Service = "github"
	return span
}
