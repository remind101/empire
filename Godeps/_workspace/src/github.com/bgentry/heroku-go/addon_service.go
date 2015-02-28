// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// Add-on services represent add-ons that may be provisioned for apps.
type AddonService struct {
	// when addon-service was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of this addon-service
	Id string `json:"id"`

	// unique name of this addon-service
	Name string `json:"name"`

	// when addon-service was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Info for existing addon-service.
//
// addonServiceIdentity is the unique identifier of the AddonService.
func (c *Client) AddonServiceInfo(addonServiceIdentity string) (*AddonService, error) {
	var addonService AddonService
	return &addonService, c.Get(&addonService, "/addon-services/"+addonServiceIdentity)
}

// List existing addon-services.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) AddonServiceList(lr *ListRange) ([]AddonService, error) {
	req, err := c.NewRequest("GET", "/addon-services", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var addonServicesRes []AddonService
	return addonServicesRes, c.DoReq(req, &addonServicesRes)
}
