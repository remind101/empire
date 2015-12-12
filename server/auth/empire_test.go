package auth

import (
	"testing"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
)

func TestAccessTokenAuthenticator(t *testing.T) {
	u := &empire.User{}
	a := &AccessTokenAuthenticator{
		findAccessToken: func(token string) (*empire.AccessToken, error) {
			assert.Equal(t, "token", token)
			return &empire.AccessToken{
				User: u,
			}, nil
		},
	}

	user, err := a.Authenticate("", "token", "")
	assert.NoError(t, err)
	assert.Equal(t, u, user)
}

func TestAccessTokenAuthenticator_TokenNotFound(t *testing.T) {
	a := &AccessTokenAuthenticator{
		findAccessToken: func(token string) (*empire.AccessToken, error) {
			assert.Equal(t, "token", token)
			return nil, nil
		},
	}

	user, err := a.Authenticate("", "token", "")
	assert.Equal(t, ErrForbidden, err)
	assert.Nil(t, user)
}
