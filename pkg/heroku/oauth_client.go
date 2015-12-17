// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// OAuth clients are applications that Heroku users can authorize to automate,
// customize or extend their usage of the platform. For more information please
// refer to the Heroku OAuth documentation.
type OAuthClient struct {
	// when OAuth client was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of this OAuth client
	Id string `json:"id"`

	// whether the client is still operable given a delinquent account
	IgnoresDelinquent *bool `json:"ignores_delinquent"`

	// OAuth client name
	Name string `json:"name"`

	// endpoint for redirection after authorization with OAuth client
	RedirectUri string `json:"redirect_uri"`

	// secret used to obtain OAuth authorizations under this client
	Secret string `json:"secret"`

	// when OAuth client was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new OAuth client.
//
// name is the OAuth client name. redirectUri is the endpoint for redirection
// after authorization with OAuth client.
func (c *Client) OAuthClientCreate(name string, redirectUri string) (*OAuthClient, error) {
	params := struct {
		Name        string `json:"name"`
		RedirectUri string `json:"redirect_uri"`
	}{
		Name:        name,
		RedirectUri: redirectUri,
	}
	var oauthClientRes OAuthClient
	return &oauthClientRes, c.Post(&oauthClientRes, "/oauth/clients", params)
}

// Delete OAuth client.
//
// oauthClientIdentity is the unique identifier of the OAuthClient.
func (c *Client) OAuthClientDelete(oauthClientIdentity string) error {
	return c.Delete("/oauth/clients/" + oauthClientIdentity)
}

// Info for an OAuth client
//
// oauthClientIdentity is the unique identifier of the OAuthClient.
func (c *Client) OAuthClientInfo(oauthClientIdentity string) (*OAuthClient, error) {
	var oauthClient OAuthClient
	return &oauthClient, c.Get(&oauthClient, "/oauth/clients/"+oauthClientIdentity)
}

// List OAuth clients
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) OAuthClientList(lr *ListRange) ([]OAuthClient, error) {
	req, err := c.NewRequest("GET", "/oauth/clients", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var oauthClientsRes []OAuthClient
	return oauthClientsRes, c.DoReq(req, &oauthClientsRes)
}

// Update OAuth client
//
// oauthClientIdentity is the unique identifier of the OAuthClient. options is
// the struct of optional parameters for this action.
func (c *Client) OAuthClientUpdate(oauthClientIdentity string, options *OAuthClientUpdateOpts) (*OAuthClient, error) {
	var oauthClientRes OAuthClient
	return &oauthClientRes, c.Patch(&oauthClientRes, "/oauth/clients/"+oauthClientIdentity, options)
}

// OAuthClientUpdateOpts holds the optional parameters for OAuthClientUpdate
type OAuthClientUpdateOpts struct {
	// OAuth client name
	Name *string `json:"name,omitempty"`
	// endpoint for redirection after authorization with OAuth client
	RedirectUri *string `json:"redirect_uri,omitempty"`
}
