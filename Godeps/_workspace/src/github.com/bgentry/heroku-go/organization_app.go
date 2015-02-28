// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// An organization app encapsulates the organization specific functionality of
// Heroku apps.
type OrganizationApp struct {
	// when app was archived
	ArchivedAt *time.Time `json:"archived_at"`

	// description from buildpack of app
	BuildpackProvidedDescription *string `json:"buildpack_provided_description"`

	// when app was created
	CreatedAt time.Time `json:"created_at"`

	// git repo URL of app
	GitURL string `json:"git_url"`

	// unique identifier of app
	Id string `json:"id"`

	// is the current member a collaborator on this app.
	Joined bool `json:"joined"`

	// are other organization members forbidden from joining this app.
	Locked bool `json:"locked"`

	// maintenance status of app
	Maintenance bool `json:"maintenance"`

	// unique name of app
	Name string `json:"name"`

	// organization that owns this app
	Organization *struct {
		Name string `json:"name"`
	} `json:"organization"`

	// identity of app owner
	Owner *struct {
		Email string `json:"email"`
		Id    string `json:"id"`
	} `json:"owner"`

	// identity of app region
	Region struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"region"`

	// when app was released
	ReleasedAt *time.Time `json:"released_at"`

	// git repo size in bytes of app
	RepoSize *int `json:"repo_size"`

	// slug size in bytes of app
	SlugSize *int `json:"slug_size"`

	// identity of app stack
	Stack struct {
		Id   string `json:"id"`
		Name string `json:"name"`
	} `json:"stack"`

	// when app was updated
	UpdatedAt time.Time `json:"updated_at"`

	// web URL of app
	WebURL string `json:"web_url"`
}

// Create a new app in the specified organization, in the default organization
// if unspecified,  or in personal account, if default organization is not set.
//
// options is the struct of optional parameters for this action.
func (c *Client) OrganizationAppCreate(options *OrganizationAppCreateOpts) (*OrganizationApp, error) {
	var organizationAppRes OrganizationApp
	return &organizationAppRes, c.Post(&organizationAppRes, "/organizations/apps", options)
}

// OrganizationAppCreateOpts holds the optional parameters for OrganizationAppCreate
type OrganizationAppCreateOpts struct {
	// are other organization members forbidden from joining this app.
	Locked *bool `json:"locked,omitempty"`
	// unique name of app
	Name *string `json:"name,omitempty"`
	// organization that owns this app
	Organization *string `json:"organization,omitempty"`
	// force creation of the app in the user account even if a default org is set.
	Personal *bool `json:"personal,omitempty"`
	// identity of app region
	Region *string `json:"region,omitempty"`
	// identity of app stack
	Stack *string `json:"stack,omitempty"`
}

// List apps in the default organization, or in personal account, if default
// organization is not set.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) OrganizationAppList(lr *ListRange) ([]OrganizationApp, error) {
	req, err := c.NewRequest("GET", "/organizations/apps", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var organizationAppsRes []OrganizationApp
	return organizationAppsRes, c.DoReq(req, &organizationAppsRes)
}

// List organization apps.
//
// organizationIdentity is the unique identifier of the OrganizationApp's
// Organization. lr is an optional ListRange that sets the Range options for the
// paginated list of results.
func (c *Client) OrganizationAppListForOrganization(organizationIdentity string, lr *ListRange) ([]OrganizationApp, error) {
	req, err := c.NewRequest("GET", "/organizations/"+organizationIdentity+"/apps", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var organizationAppsRes []OrganizationApp
	return organizationAppsRes, c.DoReq(req, &organizationAppsRes)
}

// Info for an organization app.
//
// appIdentity is the unique identifier of the OrganizationApp's App.
func (c *Client) OrganizationAppInfo(appIdentity string) (*OrganizationApp, error) {
	var organizationApp OrganizationApp
	return &organizationApp, c.Get(&organizationApp, "/organizations/apps/"+appIdentity)
}

// Lock or unlock an organization app.
//
// appIdentity is the unique identifier of the OrganizationApp's App. locked is
// the are other organization members forbidden from joining this app.
func (c *Client) OrganizationAppUpdateLocked(appIdentity string, locked bool) (*OrganizationApp, error) {
	params := struct {
		Locked bool `json:"locked"`
	}{
		Locked: locked,
	}
	var organizationAppRes OrganizationApp
	return &organizationAppRes, c.Patch(&organizationAppRes, "/organizations/apps/"+appIdentity, params)
}

// Transfer an existing organization app to another Heroku account.
//
// appIdentity is the unique identifier of the OrganizationApp's App. owner is
// the unique email address of account or unique identifier of an account.
func (c *Client) OrganizationAppTransferToAccount(appIdentity string, owner string) (*OrganizationApp, error) {
	params := struct {
		Owner string `json:"owner"`
	}{
		Owner: owner,
	}
	var organizationAppRes OrganizationApp
	return &organizationAppRes, c.Patch(&organizationAppRes, "/organizations/apps/"+appIdentity, params)
}

// Transfer an existing organization app to another organization.
//
// appIdentity is the unique identifier of the OrganizationApp's App. owner is
// the unique name of organization.
func (c *Client) OrganizationAppTransferToOrganization(appIdentity string, owner string) (*OrganizationApp, error) {
	params := struct {
		Owner string `json:"owner"`
	}{
		Owner: owner,
	}
	var organizationAppRes OrganizationApp
	return &organizationAppRes, c.Patch(&organizationAppRes, "/organizations/apps/"+appIdentity, params)
}
