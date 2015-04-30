// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// An account feature represents a Heroku labs capability that can be enabled or
// disabled for an account on Heroku.
type AccountFeature struct {
	// when account feature was created
	CreatedAt time.Time `json:"created_at"`

	// description of account feature
	Description string `json:"description"`

	// documentation URL of account feature
	DocURL string `json:"doc_url"`

	// whether or not account feature has been enabled
	Enabled bool `json:"enabled"`

	// unique identifier of account feature
	Id string `json:"id"`

	// unique name of account feature
	Name string `json:"name"`

	// state of account feature
	State string `json:"state"`

	// when account feature was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Info for an existing account feature.
//
// accountFeatureIdentity is the unique identifier of the AccountFeature.
func (c *Client) AccountFeatureInfo(accountFeatureIdentity string) (*AccountFeature, error) {
	var accountFeature AccountFeature
	return &accountFeature, c.Get(&accountFeature, "/account/features/"+accountFeatureIdentity)
}

// List existing account features.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) AccountFeatureList(lr *ListRange) ([]AccountFeature, error) {
	req, err := c.NewRequest("GET", "/account/features", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var accountFeaturesRes []AccountFeature
	return accountFeaturesRes, c.DoReq(req, &accountFeaturesRes)
}

// Update an existing account feature.
//
// accountFeatureIdentity is the unique identifier of the AccountFeature.
// enabled is the whether or not account feature has been enabled.
func (c *Client) AccountFeatureUpdate(accountFeatureIdentity string, enabled bool) (*AccountFeature, error) {
	params := struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: enabled,
	}
	var accountFeatureRes AccountFeature
	return &accountFeatureRes, c.Patch(&accountFeatureRes, "/account/features/"+accountFeatureIdentity, params)
}
