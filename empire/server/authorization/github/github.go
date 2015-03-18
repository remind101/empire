package github

import (
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server/authorization"
)

// Authorizer is an implementation of the authorization.Authorizer interface backed by
// GitHub's Non-Web Application Flow, which can be found at
// http://goo.gl/onpQKM.
type Authorizer struct {
	// OAuth scopes that should be granted with the access token.
	Scopes []string

	// The oauth application client id.
	ClientID string

	// The oauth application client secret.
	ClientSecret string

	// If provided, it will ensure that the user is a member of this
	// organization.
	Organization string

	client interface {
		CreateAuthorization(CreateAuthorizationOpts) (*Authorization, error)
		GetUser(token string) (*User, error)
		IsMember(organization, token string) (bool, error)
	}
}

func (a *Authorizer) Authorize(username, password, twofactor string) (*empire.User, error) {
	c := a.client
	if c == nil {
		c = &Client{}
	}

	auth, err := c.CreateAuthorization(CreateAuthorizationOpts{
		Scopes:       a.Scopes,
		ClientID:     a.ClientID,
		ClientSecret: a.ClientSecret,
		Username:     username,
		Password:     password,
		TwoFactor:    twofactor,
	})
	if err != nil {
		if err == errTwoFactor {
			return nil, authorization.ErrTwoFactor
		}

		return nil, authorization.ErrUnauthorized
	}

	u, err := c.GetUser(auth.Token)
	if err != nil {
		return nil, err
	}

	if a.Organization != "" {
		ok, err := c.IsMember(a.Organization, auth.Token)
		if err != nil {
			return nil, err
		}

		if !ok {
			return nil, &authorization.MembershipError{
				Organization: a.Organization,
			}
		}
	}

	return &empire.User{
		Name:        u.Login,
		GitHubToken: auth.Token,
	}, nil
}
