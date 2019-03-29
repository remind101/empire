package github

import (
	"testing"

	"github.com/remind101/empire"
	"github.com/remind101/empire/server/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestAuthenticator(t *testing.T) {
	c := new(mockClient)
	a := &Authenticator{client: c}

	c.On("CreateAuthorization", CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
		OTP:      "otp",
	}).Return(&Authorization{
		Token: "access_token",
	}, nil)

	c.On("GetUser", "access_token").Return(&User{
		Login: "ejholmes",
	}, nil)

	c.On("GetPrimaryEmail", "access_token").Return(&Email{
		Email: "eric@example.org",
	}, nil)

	session, err := a.Authenticate("username", "password", "otp")
	assert.NoError(t, err)
	assert.Equal(t, "ejholmes", session.User.Name)
	assert.Equal(t, "access_token", session.User.GitHubToken)
}

func TestAuthenticator_FailFast(t *testing.T) {
	a := &Authenticator{}

	_, err := a.Authenticate("username", "", "")
	assert.Equal(t, err, auth.ErrForbidden)

	_, err = a.Authenticate("", "password", "")
	assert.Equal(t, err, auth.ErrForbidden)
}

func TestAuthenticator_ErrTwoFactor(t *testing.T) {
	c := new(mockClient)
	a := &Authenticator{client: c}

	c.On("CreateAuthorization", CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
	}).Return(nil, errTwoFactor)

	session, err := a.Authenticate("username", "password", "")
	assert.Equal(t, auth.ErrTwoFactor, err)
	assert.Nil(t, session)
}

func TestAuthenticator_ErrForbidden(t *testing.T) {
	c := new(mockClient)
	a := &Authenticator{client: c}

	c.On("CreateAuthorization", CreateAuthorizationOptions{
		Username: "username",
		Password: "badpassword",
	}).Return(nil, errUnauthorized)

	session, err := a.Authenticate("username", "badpassword", "")
	assert.Equal(t, auth.ErrForbidden, err)
	assert.Nil(t, session)
}

func TestOrganizationAuthorizer(t *testing.T) {
	c := new(mockClient)
	a := &OrganizationAuthorizer{
		Organization: "remind101",
		client:       c,
	}

	c.On("IsOrganizationMember", "remind101", "access_token").Return(true, nil)

	err := a.Authorize(&empire.User{GitHubToken: "access_token"})
	assert.NoError(t, err)
}

func TestOrganizationAuthorizer_Unauthorized(t *testing.T) {
	c := new(mockClient)
	a := &OrganizationAuthorizer{
		Organization: "remind101",
		client:       c,
	}

	c.On("IsOrganizationMember", "remind101", "access_token").Return(false, nil)

	err := a.Authorize(&empire.User{
		Name:        "ejholmes",
		GitHubToken: "access_token"},
	)
	assert.IsType(t, &auth.UnauthorizedError{}, err)
	assert.EqualError(t, err, `ejholmes is not a member of the "remind101" organization.`)
}

func TestTeamAuthorizer(t *testing.T) {
	c := new(mockClient)
	a := &TeamAuthorizer{
		TeamID: "123",
		client: c,
	}

	c.On("IsTeamMember", "123", "access_token").Return(true, nil)

	err := a.Authorize(&empire.User{
		Name:        "ejholmes",
		GitHubToken: "access_token",
	})
	assert.NoError(t, err)
}

func TestTeamAuthorizer_Unauthorized(t *testing.T) {
	c := new(mockClient)
	a := &TeamAuthorizer{
		TeamID: "123",
		client: c,
	}

	c.On("IsTeamMember", "123", "access_token").Return(false, nil)

	err := a.Authorize(&empire.User{
		Name:        "ejholmes",
		GitHubToken: "access_token",
	})
	assert.IsType(t, &auth.UnauthorizedError{}, err)
	assert.EqualError(t, err, `ejholmes is not a member of team 123.`)
}

type mockClient struct {
	mock.Mock
}

func (m *mockClient) CreateAuthorization(opts CreateAuthorizationOptions) (*Authorization, error) {
	args := m.Called(opts)
	auth := args.Get(0)
	if auth != nil {
		return auth.(*Authorization), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockClient) GetUser(token string) (*User, error) {
	args := m.Called(token)
	user := args.Get(0)
	if user != nil {
		return user.(*User), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockClient) GetPrimaryEmail(token string) (*Email, error) {
	args := m.Called(token)
	email := args.Get(0)
	if email != nil {
		return email.(*Email), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *mockClient) IsOrganizationMember(organization, token string) (bool, error) {
	args := m.Called(organization, token)
	return args.Bool(0), args.Error(1)
}

func (m *mockClient) IsTeamMember(teamID, token string) (bool, error) {
	args := m.Called(teamID, token)
	return args.Bool(0), args.Error(1)
}
