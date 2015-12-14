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

	user, err := a.Authenticate("username", "password", "otp")
	assert.NoError(t, err)
	assert.Equal(t, "ejholmes", user.Name)
	assert.Equal(t, "access_token", user.GitHubToken)
}

func TestAuthenticator_ErrTwoFactor(t *testing.T) {
	c := new(mockClient)
	a := &Authenticator{client: c}

	c.On("CreateAuthorization", CreateAuthorizationOptions{
		Username: "username",
		Password: "password",
	}).Return(nil, errTwoFactor)

	user, err := a.Authenticate("username", "password", "")
	assert.Equal(t, auth.ErrTwoFactor, err)
	assert.Nil(t, user)
}

func TestAuthenticator_ErrForbidden(t *testing.T) {
	c := new(mockClient)
	a := &Authenticator{client: c}

	c.On("CreateAuthorization", CreateAuthorizationOptions{
		Username: "username",
		Password: "badpassword",
	}).Return(nil, errUnauthorized)

	user, err := a.Authenticate("username", "badpassword", "")
	assert.Equal(t, auth.ErrForbidden, err)
	assert.Nil(t, user)
}

func TestOrganizationAuthorizer(t *testing.T) {
	c := new(mockClient)
	a := &OrganizationAuthorizer{
		Organization: "remind101",
		client:       c,
	}

	c.On("IsMember", "remind101", "access_token").Return(true, nil)

	err := a.Authorize(&empire.User{GitHubToken: "access_token"})
	assert.NoError(t, err)
}

func TestOrganizationAuthorizer_Unauthorized(t *testing.T) {
	c := new(mockClient)
	a := &OrganizationAuthorizer{
		Organization: "remind101",
		client:       c,
	}

	c.On("IsMember", "remind101", "access_token").Return(false, nil)

	err := a.Authorize(&empire.User{
		Name:        "ejholmes",
		GitHubToken: "access_token"},
	)
	assert.IsType(t, &auth.UnauthorizedError{}, err)
	assert.EqualError(t, err, `ejholmes is not a member of the "remind101" organization`)
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

func (m *mockClient) IsMember(organization, token string) (bool, error) {
	args := m.Called(organization, token)
	return args.Bool(0), args.Error(1)
}
