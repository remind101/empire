// Package github provides auth.Authentication and auth.Authorizer
// implementations backed by GitHub users, orgs and teams.
package github

import (
	"fmt"
	"regexp"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
)

// DefaultIDFunc is an IDFunc that just returns the users GH username as the
// user id.
var DefaultIDFunc = func(ghUser *User) (string, error) {
	return ghUser.Login, nil
}

var emailRegex = regexp.MustCompile(`.+@(.+)`)

// EmailDomainID returns an IDFunc that returns the first email for the given
// domain.
func EmailDomainID(domain string) func(*User) (string, error) {
	return func(ghUser *User) (string, error) {
		for _, email := range ghUser.Emails {
			matches := emailRegex.FindAllStringSubmatch(email, -1)
			if matches[0][1] == domain {
				return email, nil
			}
		}

		return "", fmt.Errorf("GitHub user %s has no @%s email", ghUser.Login, domain)
	}
}

// Authorizer is an implementation of the auth.Authenticator interface backed by
// GitHub's Non-Web Application Flow, which can be found at
// http://goo.gl/onpQKM.
type Authenticator struct {
	// IDFunc should return something that can be used as a canonical user
	// id within Empire (for example, their company email).
	IDFunc func(*User) (string, error)

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

	idFunc := a.IDFunc
	if idFunc == nil {
		idFunc = DefaultIDFunc
	}
	id, err := idFunc(u)
	if err != nil {
		return nil, fmt.Errorf("could not determine user id: %v", err)
	}

	return &empire.User{
		ID:          id,
		Name:        u.Login,
		GitHubToken: authorization.Token,
	}, nil
}

// OrganizationAuthorizer is an implementation of the auth.Authorizer interface
// that checks that the user is a member of the given GitHub organization.
type OrganizationAuthorizer struct {
	Organization string

	client interface {
		IsOrganizationMember(organization, token string) (bool, error)
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

	ok, err := a.client.IsOrganizationMember(a.Organization, user.GitHubToken)
	if err != nil {
		return err
	}

	if !ok {
		return &auth.UnauthorizedError{
			Reason: fmt.Sprintf("%s is not a member of the \"%s\" organization.", user.Name, a.Organization),
		}
	}

	return nil
}

// TeamAuthorizer is an implementation of the auth.Authorizer interface that
// checks that the user is a member of the given GitHub team.
type TeamAuthorizer struct {
	TeamID string

	client interface {
		IsTeamMember(teamID, token string) (bool, error)
	}
}

func NewTeamAuthorizer(c *Client) *TeamAuthorizer {
	return &TeamAuthorizer{client: c}
}

func (a *TeamAuthorizer) Authorize(user *empire.User) error {
	if a.TeamID == "" {
		panic("no team id set")
	}

	ok, err := a.client.IsTeamMember(a.TeamID, user.GitHubToken)
	if err != nil {
		return err
	}

	if !ok {
		return &auth.UnauthorizedError{
			Reason: fmt.Sprintf("%s is not a member of team %s.", user.Name, a.TeamID),
		}
	}

	return nil
}
