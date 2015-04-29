// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// OAuth authorizations represent clients that a Heroku user has authorized to
// automate, customize or extend their usage of the platform. For more
// information please refer to the Heroku OAuth documentation
type OAuthAuthorization struct {
	// access token for this authorization
	AccessToken *struct {
		ExpiresIn *int   `json:"expires_in"`
		Id        string `json:"id"`
		Token     string `json:"token"`
	} `json:"access_token"`

	// identifier of the client that obtained this authorization, if any
	Client *struct {
		Id          string `json:"id"`
		Name        string `json:"name"`
		RedirectUri string `json:"redirect_uri"`
	} `json:"client"`

	// when OAuth authorization was created
	CreatedAt time.Time `json:"created_at"`

	// this authorization's grant
	Grant *struct {
		Code      string `json:"code"`
		ExpiresIn int    `json:"expires_in"`
		Id        string `json:"id"`
	} `json:"grant"`

	// unique identifier of OAuth authorization
	Id string `json:"id"`

	// refresh token for this authorization
	RefreshToken *struct {
		ExpiresIn *int   `json:"expires_in"`
		Id        string `json:"id"`
		Token     string `json:"token"`
	} `json:"refresh_token"`

	// The scope of access OAuth authorization allows
	Scope []string `json:"scope"`

	// when OAuth authorization was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new OAuth authorization.
//
// scope is the The scope of access OAuth authorization allows. options is the
// struct of optional parameters for this action.
func (c *Client) OAuthAuthorizationCreate(scope []string, options *OAuthAuthorizationCreateOpts) (*OAuthAuthorization, error) {
	params := struct {
		Scope       []string `json:"scope"`
		Client      *string  `json:"client,omitempty"`
		Description *string  `json:"description,omitempty"`
		ExpiresIn   *int     `json:"expires_in,omitempty"`
	}{
		Scope: scope,
	}
	if options != nil {
		params.Client = options.Client
		params.Description = options.Description
		params.ExpiresIn = options.ExpiresIn
	}
	var oauthAuthorizationRes OAuthAuthorization
	return &oauthAuthorizationRes, c.Post(&oauthAuthorizationRes, "/oauth/authorizations", params)
}

// OAuthAuthorizationCreateOpts holds the optional parameters for OAuthAuthorizationCreate
type OAuthAuthorizationCreateOpts struct {
	// identifier of the client that obtained this authorization, if any
	Client *string `json:"client,omitempty"`
	// human-friendly description of this OAuth authorization
	Description *string `json:"description,omitempty"`
	// seconds until OAuth token expires; may be `null` for tokens with indefinite lifetime
	ExpiresIn *int `json:"expires_in,omitempty"`
}

// Delete OAuth authorization.
//
// oauthAuthorizationIdentity is the unique identifier of the
// OAuthAuthorization.
func (c *Client) OAuthAuthorizationDelete(oauthAuthorizationIdentity string) error {
	return c.Delete("/oauth/authorizations/" + oauthAuthorizationIdentity)
}

// Info for an OAuth authorization.
//
// oauthAuthorizationIdentity is the unique identifier of the
// OAuthAuthorization.
func (c *Client) OAuthAuthorizationInfo(oauthAuthorizationIdentity string) (*OAuthAuthorization, error) {
	var oauthAuthorization OAuthAuthorization
	return &oauthAuthorization, c.Get(&oauthAuthorization, "/oauth/authorizations/"+oauthAuthorizationIdentity)
}

// List OAuth authorizations.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) OAuthAuthorizationList(lr *ListRange) ([]OAuthAuthorization, error) {
	req, err := c.NewRequest("GET", "/oauth/authorizations", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var oauthAuthorizationsRes []OAuthAuthorization
	return oauthAuthorizationsRes, c.DoReq(req, &oauthAuthorizationsRes)
}
