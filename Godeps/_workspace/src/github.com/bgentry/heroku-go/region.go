// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// A region represents a geographic location in which your application may run.
type Region struct {
	// when region was created
	CreatedAt time.Time `json:"created_at"`

	// description of region
	Description string `json:"description"`

	// unique identifier of region
	Id string `json:"id"`

	// unique name of region
	Name string `json:"name"`

	// when region was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Info for existing region.
//
// regionIdentity is the unique identifier of the Region.
func (c *Client) RegionInfo(regionIdentity string) (*Region, error) {
	var region Region
	return &region, c.Get(&region, "/regions/"+regionIdentity)
}

// List existing regions.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) RegionList(lr *ListRange) ([]Region, error) {
	req, err := c.NewRequest("GET", "/regions", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var regionsRes []Region
	return regionsRes, c.DoReq(req, &regionsRes)
}
