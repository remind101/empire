// Package github provides auth.Authentication and auth.Authorizer
// implementations backed by GitHub users, orgs and teams.
package github

import (
	"fmt"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
)

// Authorizer is an implementation of the auth.Authenticator interface backed by
// GitHub's Non-Web Application Flow, which can be found at
// http://goo.gl/onpQKM.
type Authenticator struct {
	// OAuth2 configuration (client id, secret, scopes, etc).
	client interface {
		CreateAuthorization(CreateAuthorizationOptions) (*Authorization, error)
		GetUser(token string) (*User, error)
	}
}

// NewAuthenticator returns a new Authenticator instance that uses the given
// Client to make calls to GitHub.
func NewAuthenticator(c *Client) *Authenticator {
	return &Authenticator{client: c}
}

func (a *Authenticator) Authenticate(username, password, otp string) (*empire.User, error) {
	authorization, err := a.client.CreateAuthorization(CreateAuthorizationOptions{
		Username: username,
		Password: password,
		OTP:      otp,
	})
	if err != nil {
		switch err {
		case errTwoFactor:
			return nil, auth.ErrTwoFactor
		case errUnauthorized:
			return nil, auth.ErrForbidden
		default:
			return nil, err
		}
	}

	u, err := a.client.GetUser(authorization.Token)
	if err != nil {
		return nil, err
	}

	return &empire.User{
		Name:        u.Login,
		GitHubToken: authorization.Token,
	}, nil
}

// OrganizationAuthorizer is an implementation of the auth.Authorizer interface
// that checks that the user is a member of the given GitHub organization.
type OrganizationAuthorizer struct {
	Organization string

	client interface {
		IsMember(organization, token string) (bool, error)
	}
}

// NewOrganizationAuthorizer returns a new OrganizationAuthorizer instance.
func NewOrganizationAuthorizer(c *Client) *OrganizationAuthorizer {
	return &OrganizationAuthorizer{client: c}
}

func (a *OrganizationAuthorizer) Authorize(user *empire.User) error {
	if a.Organization == "" {
		// Probably a configuration error
		panic("no organization set")
	}

	ok, err := a.client.IsMember(a.Organization, user.GitHubToken)
	if err != nil {
		return err
	}

	if !ok {
		return &auth.UnauthorizedError{
			Reason: fmt.Sprintf("%s is not a member of the \"%s\" organization", user.Name, a.Organization),
		}
	}

	return nil
}
