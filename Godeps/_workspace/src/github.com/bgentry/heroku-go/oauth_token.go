// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// OAuth tokens provide access for authorized clients to act on behalf of a
// Heroku user to automate, customize or extend their usage of the platform. For
// more information please refer to the Heroku OAuth documentation
type OAuthToken struct {
	// current access token
	AccessToken struct {
		ExpiresIn *int   `json:"expires_in"`
		Id        string `json:"id"`
		Token     string `json:"token"`
	} `json:"access_token"`

	// authorization for this set of tokens
	Authorization struct {
		Id string `json:"id"`
	} `json:"authorization"`

	// OAuth client secret used to obtain token
	Client *struct {
		Secret string `json:"secret"`
	} `json:"client"`

	// when OAuth token was created
	CreatedAt time.Time `json:"created_at"`

	// grant used on the underlying authorization
	Grant struct {
		Code string `json:"code"`
		Type string `json:"type"`
	} `json:"grant"`

	// unique identifier of OAuth token
	Id string `json:"id"`

	// refresh token for this authorization
	RefreshToken struct {
		ExpiresIn *int   `json:"expires_in"`
		Id        string `json:"id"`
		Token     string `json:"token"`
	} `json:"refresh_token"`

	// OAuth session using this token
	Session struct {
		Id string `json:"id"`
	} `json:"session"`

	// when OAuth token was updated
	UpdatedAt time.Time `json:"updated_at"`

	// Reference to the user associated with this token
	User struct {
		Id string `json:"id"`
	} `json:"user"`
}

// Create a new OAuth token.
//
// grant is the grant used on the underlying authorization. client is the OAuth
// client secret used to obtain token. refreshToken is the refresh token for
// this authorization.
func (c *Client) OAuthTokenCreate(grant OAuthTokenCreateGrant, client OAuthTokenCreateClient, refreshToken OAuthTokenCreateRefreshToken) (*OAuthToken, error) {
	params := struct {
		Grant        OAuthTokenCreateGrant        `json:"grant"`
		Client       OAuthTokenCreateClient       `json:"client"`
		RefreshToken OAuthTokenCreateRefreshToken `json:"refresh_token"`
	}{
		Grant:        grant,
		Client:       client,
		RefreshToken: refreshToken,
	}
	var oauthTokenRes OAuthToken
	return &oauthTokenRes, c.Post(&oauthTokenRes, "/oauth/tokens", params)
}

// OAuthTokenCreateGrant used in OAuthTokenCreate as the grant used on the underlying authorization
type OAuthTokenCreateGrant struct {
	// grant code received from OAuth web application authorization
	Code string `json:"code"`

	// type of grant requested, one of `authorization_code` or `refresh_token`
	Type string `json:"type"`
}

// OAuthTokenCreateClient used in OAuthTokenCreate as the OAuth client secret used to obtain token
type OAuthTokenCreateClient struct {
	// secret used to obtain OAuth authorizations under this client
	Secret string `json:"secret"`
}

// OAuthTokenCreateRefreshToken used in OAuthTokenCreate as the refresh token for this authorization
type OAuthTokenCreateRefreshToken struct {
	// contents of the token to be used for authorization
	Token string `json:"token"`
}
