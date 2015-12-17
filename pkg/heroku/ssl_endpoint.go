// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// SSL Endpoint is a public address serving custom SSL cert for HTTPS traffic to
// a Heroku app. Note that an app must have the ssl:endpoint addon installed
// before it can provision an SSL Endpoint using these APIs.
type SSLEndpoint struct {
	// raw contents of the public certificate chain (eg: .crt or .pem file)
	CertificateChain string `json:"certificate_chain"`

	// canonical name record, the address to point a domain at
	Cname string `json:"cname"`

	// when endpoint was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of this SSL endpoint
	Id string `json:"id"`

	// unique name for SSL endpoint
	Name string `json:"name"`

	// when endpoint was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new SSL endpoint.
//
// appIdentity is the unique identifier of the SSLEndpoint's App.
// certificateChain is the raw contents of the public certificate chain (eg:
// .crt or .pem file). privateKey is the contents of the private key (eg .key
// file). options is the struct of optional parameters for this action.
func (c *Client) SSLEndpointCreate(appIdentity string, certificateChain string, privateKey string, options *SSLEndpointCreateOpts) (*SSLEndpoint, error) {
	params := struct {
		CertificateChain string `json:"certificate_chain"`
		PrivateKey       string `json:"private_key"`
		Preprocess       *bool  `json:"preprocess,omitempty"`
	}{
		CertificateChain: certificateChain,
		PrivateKey:       privateKey,
	}
	if options != nil {
		params.Preprocess = options.Preprocess
	}
	var sslEndpointRes SSLEndpoint
	return &sslEndpointRes, c.Post(&sslEndpointRes, "/apps/"+appIdentity+"/ssl-endpoints", params)
}

// SSLEndpointCreateOpts holds the optional parameters for SSLEndpointCreate
type SSLEndpointCreateOpts struct {
	// allow Heroku to modify an uploaded public certificate chain if deemed advantageous by adding missing intermediaries, stripping unnecessary ones, etc.
	Preprocess *bool `json:"preprocess,omitempty"`
}

// Delete existing SSL endpoint.
//
// appIdentity is the unique identifier of the SSLEndpoint's App.
// sslEndpointIdentity is the unique identifier of the SSLEndpoint.
func (c *Client) SSLEndpointDelete(appIdentity string, sslEndpointIdentity string) error {
	return c.Delete("/apps/" + appIdentity + "/ssl-endpoints/" + sslEndpointIdentity)
}

// Info for existing SSL endpoint.
//
// appIdentity is the unique identifier of the SSLEndpoint's App.
// sslEndpointIdentity is the unique identifier of the SSLEndpoint.
func (c *Client) SSLEndpointInfo(appIdentity string, sslEndpointIdentity string) (*SSLEndpoint, error) {
	var sslEndpoint SSLEndpoint
	return &sslEndpoint, c.Get(&sslEndpoint, "/apps/"+appIdentity+"/ssl-endpoints/"+sslEndpointIdentity)
}

// List existing SSL endpoints.
//
// appIdentity is the unique identifier of the SSLEndpoint's App. lr is an
// optional ListRange that sets the Range options for the paginated list of
// results.
func (c *Client) SSLEndpointList(appIdentity string, lr *ListRange) ([]SSLEndpoint, error) {
	req, err := c.NewRequest("GET", "/apps/"+appIdentity+"/ssl-endpoints", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var sslEndpointsRes []SSLEndpoint
	return sslEndpointsRes, c.DoReq(req, &sslEndpointsRes)
}

// Update an existing SSL endpoint.
//
// appIdentity is the unique identifier of the SSLEndpoint's App.
// sslEndpointIdentity is the unique identifier of the SSLEndpoint. options is
// the struct of optional parameters for this action.
func (c *Client) SSLEndpointUpdate(appIdentity string, sslEndpointIdentity string, options *SSLEndpointUpdateOpts) (*SSLEndpoint, error) {
	var sslEndpointRes SSLEndpoint
	return &sslEndpointRes, c.Patch(&sslEndpointRes, "/apps/"+appIdentity+"/ssl-endpoints/"+sslEndpointIdentity, options)
}

// SSLEndpointUpdateOpts holds the optional parameters for SSLEndpointUpdate
type SSLEndpointUpdateOpts struct {
	// raw contents of the public certificate chain (eg: .crt or .pem file)
	CertificateChain *string `json:"certificate_chain,omitempty"`
	// allow Heroku to modify an uploaded public certificate chain if deemed advantageous by adding missing intermediaries, stripping unnecessary ones, etc.
	Preprocess *bool `json:"preprocess,omitempty"`
	// contents of the private key (eg .key file)
	PrivateKey *string `json:"private_key,omitempty"`
	// indicates that a rollback should be performed
	Rollback *bool `json:"rollback,omitempty"`
}
