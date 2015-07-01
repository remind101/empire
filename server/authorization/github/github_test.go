package github

import (
	"testing"

	"github.com/remind101/empire/server/authorization"
)

func TestAuthorizeTwoFactorRequired(t *testing.T) {
	c := &mockClient{
		CreateAuthorizationFunc: func(opts CreateAuthorizationOpts) (*Authorization, error) {
			return nil, errTwoFactor
		},
	}
	a := &Authorizer{client: c}

	_, err := a.Authorize("", "", "")

	if err != authorization.ErrTwoFactor {
		t.Fatalf("err => %v; want %v", err, authorization.ErrTwoFactor)
	}
}

func TestAuthorizeOrganization(t *testing.T) {
	org := "remind101"

	c := &mockClient{
		CreateAuthorizationFunc: func(opts CreateAuthorizationOpts) (*Authorization, error) {
			return &Authorization{}, nil
		},
		GetUserFunc: func(token string) (*User, error) {
			return &User{}, nil
		},
		IsMemberFunc: func(organization, token string) (bool, error) {
			if got, want := organization, org; got != want {
				t.Fatalf("Organization => %s; want %s", got, want)
			}

			return false, nil
		},
	}
	a := &Authorizer{
		Organization: org,
		client:       c,
	}

	_, err := a.Authorize("", "", "")
	if _, ok := err.(*authorization.MembershipError); !ok {
		t.Fatalf("err => %v; want a membership error", err)
	}
}

type mockClient struct {
	CreateAuthorizationFunc func(CreateAuthorizationOpts) (*Authorization, error)
	GetUserFunc             func(token string) (*User, error)
	IsMemberFunc            func(organization, token string) (bool, error)
}

func (c *mockClient) CreateAuthorization(opts CreateAuthorizationOpts) (*Authorization, error) {
	return c.CreateAuthorizationFunc(opts)
}

func (c *mockClient) GetUser(token string) (*User, error) {
	return c.GetUserFunc(token)
}

func (c *mockClient) IsMember(organization, token string) (bool, error) {
	return c.IsMemberFunc(organization, token)
}
