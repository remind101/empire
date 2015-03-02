// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// A release represents a combination of code, config vars and add-ons for an
// app on Heroku.
type Release struct {
	// when release was created
	CreatedAt time.Time `json:"created_at"`

	// description of changes in this release
	Description string `json:"description"`

	// unique identifier of release
	Id string `json:"id"`

	// when release was updated
	UpdatedAt time.Time `json:"updated_at"`

	// slug running in this release
	Slug *struct {
		Id string `json:"id"`
	} `json:"slug"`

	// user that created the release
	User struct {
		Id    string `json:"id"`
		Email string `json:"email"`
	} `json:"user"`

	// unique version assigned to the release
	Version int `json:"version"`
}

// Info for existing release.
//
// appIdentity is the unique identifier of the Release's App. releaseIdentity is
// the unique identifier of the Release.
func (c *Client) ReleaseInfo(appIdentity string, releaseIdentity string) (*Release, error) {
	var release Release
	return &release, c.Get(&release, "/apps/"+appIdentity+"/releases/"+releaseIdentity)
}

// List existing releases.
//
// appIdentity is the unique identifier of the Release's App. lr is an optional
// ListRange that sets the Range options for the paginated list of results.
func (c *Client) ReleaseList(appIdentity string, lr *ListRange) ([]Release, error) {
	req, err := c.NewRequest("GET", "/apps/"+appIdentity+"/releases", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var releasesRes []Release
	return releasesRes, c.DoReq(req, &releasesRes)
}

// Create new release. The API cannot be used to create releases on Bamboo apps.
//
// appIdentity is the unique identifier of the Release's App. slug is the unique
// identifier of slug. options is the struct of optional parameters for this
// action.
func (c *Client) ReleaseCreate(appIdentity string, slug string, options *ReleaseCreateOpts) (*Release, error) {
	params := struct {
		Slug        string  `json:"slug"`
		Description *string `json:"description,omitempty"`
	}{
		Slug: slug,
	}
	if options != nil {
		params.Description = options.Description
	}
	var releaseRes Release
	return &releaseRes, c.Post(&releaseRes, "/apps/"+appIdentity+"/releases", params)
}

// ReleaseCreateOpts holds the optional parameters for ReleaseCreate
type ReleaseCreateOpts struct {
	// description of changes in this release
	Description *string `json:"description,omitempty"`
}

// Rollback to an existing release.
//
// appIdentity is the unique identifier of the Release's App. release is the
// unique identifier of release.
func (c *Client) ReleaseRollback(appIdentity string, release string) (*Release, error) {
	params := struct {
		Release string `json:"release"`
	}{
		Release: release,
	}
	var releaseRes Release
	return &releaseRes, c.Post(&releaseRes, "/apps/"+appIdentity+"/releases", params)
}
