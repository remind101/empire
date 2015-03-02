// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// An app represents the program that you would like to deploy and run on
// Heroku.
type App struct {
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

	// maintenance status of app
	Maintenance bool `json:"maintenance"`

	// unique name of app
	Name string `json:"name"`

	// identity of app owner
	Owner struct {
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

// Create a new app.
//
// options is the struct of optional parameters for this action.
func (c *Client) AppCreate(options *AppCreateOpts) (*App, error) {
	var appRes App
	return &appRes, c.Post(&appRes, "/apps", options)
}

// AppCreateOpts holds the optional parameters for AppCreate
type AppCreateOpts struct {
	// unique name of app
	Name *string `json:"name,omitempty"`
	// identity of app region
	Region *string `json:"region,omitempty"`
	// identity of app stack
	Stack *string `json:"stack,omitempty"`
}

// Delete an existing app.
//
// appIdentity is the unique identifier of the App.
func (c *Client) AppDelete(appIdentity string) error {
	return c.Delete("/apps/" + appIdentity)
}

// Info for existing app.
//
// appIdentity is the unique identifier of the App.
func (c *Client) AppInfo(appIdentity string) (*App, error) {
	var app App
	return &app, c.Get(&app, "/apps/"+appIdentity)
}

// List existing apps.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) AppList(lr *ListRange) ([]App, error) {
	req, err := c.NewRequest("GET", "/apps", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var appsRes []App
	return appsRes, c.DoReq(req, &appsRes)
}

// Update an existing app.
//
// appIdentity is the unique identifier of the App. options is the struct of
// optional parameters for this action.
func (c *Client) AppUpdate(appIdentity string, options *AppUpdateOpts) (*App, error) {
	var appRes App
	return &appRes, c.Patch(&appRes, "/apps/"+appIdentity, options)
}

// AppUpdateOpts holds the optional parameters for AppUpdate
type AppUpdateOpts struct {
	// maintenance status of app
	Maintenance *bool `json:"maintenance,omitempty"`
	// unique name of app
	Name *string `json:"name,omitempty"`
}
