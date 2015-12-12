package auth

import "github.com/remind101/empire"

// AccessTokenAuthenticator is an Authenticator that uses empire JWT access tokens to
// authenticate.
type AccessTokenAuthenticator struct {
	findAccessToken func(string) (*empire.AccessToken, error)
}

// NewAccessTokenAuthenticator reutrns a new AccessTokenAuthenticator.
func NewAccessTokenAuthenticator(e *empire.Empire) *AccessTokenAuthenticator {
	return &AccessTokenAuthenticator{findAccessToken: e.AccessTokensFind}
}

// Authenticate authenticates the access token, which should be provided as the
// password parameter. Username and otp are ignored.
func (a *AccessTokenAuthenticator) Authenticate(_ string, token string, _ string) (*empire.User, error) {
	at, err := a.findAccessToken(token)
	if err != nil {
		return nil, err
	}

	if at == nil {
		return nil, ErrForbidden
	}

	return at.User, nil
}
