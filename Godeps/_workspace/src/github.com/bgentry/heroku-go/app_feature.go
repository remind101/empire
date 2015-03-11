// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// An app feature represents a Heroku labs capability that can be enabled or
// disabled for an app on Heroku.
type AppFeature struct {
	// when app feature was created
	CreatedAt time.Time `json:"created_at"`

	// description of app feature
	Description string `json:"description"`

	// documentation URL of app feature
	DocURL string `json:"doc_url"`

	// whether or not app feature has been enabled
	Enabled bool `json:"enabled"`

	// unique identifier of app feature
	Id string `json:"id"`

	// unique name of app feature
	Name string `json:"name"`

	// state of app feature
	State string `json:"state"`

	// when app feature was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Info for an existing app feature.
//
// appIdentity is the unique identifier of the AppFeature's App.
// appFeatureIdentity is the unique identifier of the AppFeature.
func (c *Client) AppFeatureInfo(appIdentity string, appFeatureIdentity string) (*AppFeature, error) {
	var appFeature AppFeature
	return &appFeature, c.Get(&appFeature, "/apps/"+appIdentity+"/features/"+appFeatureIdentity)
}

// List existing app features.
//
// appIdentity is the unique identifier of the AppFeature's App. lr is an
// optional ListRange that sets the Range options for the paginated list of
// results.
func (c *Client) AppFeatureList(appIdentity string, lr *ListRange) ([]AppFeature, error) {
	req, err := c.NewRequest("GET", "/apps/"+appIdentity+"/features", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var appFeaturesRes []AppFeature
	return appFeaturesRes, c.DoReq(req, &appFeaturesRes)
}

// Update an existing app feature.
//
// appIdentity is the unique identifier of the AppFeature's App.
// appFeatureIdentity is the unique identifier of the AppFeature. enabled is the
// whether or not app feature has been enabled.
func (c *Client) AppFeatureUpdate(appIdentity string, appFeatureIdentity string, enabled bool) (*AppFeature, error) {
	params := struct {
		Enabled bool `json:"enabled"`
	}{
		Enabled: enabled,
	}
	var appFeatureRes AppFeature
	return &appFeatureRes, c.Patch(&appFeatureRes, "/apps/"+appIdentity+"/features/"+appFeatureIdentity, params)
}
