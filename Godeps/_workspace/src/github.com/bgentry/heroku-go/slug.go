// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// A slug is a snapshot of your application code that is ready to run on the
// platform.
type Slug struct {
	// pointer to the url where clients can fetch or store the actual release binary
	Blob struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"blob"`

	// description from buildpack of slug
	BuildpackProvidedDescription *string `json:"buildpack_provided_description"`

	// identification of the code with your version control system (eg: SHA of the git HEAD)
	Commit *string `json:"commit"`

	// when slug was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of slug
	Id string `json:"id"`

	// hash mapping process type names to their respective command
	ProcessTypes map[string]string `json:"process_types"`

	// size of slug, in bytes
	Size *int `json:"size"`

	// when slug was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Info for existing slug.
//
// appIdentity is the unique identifier of the Slug's App. slugIdentity is the
// unique identifier of the Slug.
func (c *Client) SlugInfo(appIdentity string, slugIdentity string) (*Slug, error) {
	var slug Slug
	return &slug, c.Get(&slug, "/apps/"+appIdentity+"/slugs/"+slugIdentity)
}

// Create a new slug. For more information please refer to Deploying Slugs using
// the Platform API.
//
// appIdentity is the unique identifier of the Slug's App. processTypes is the
// hash mapping process type names to their respective command. options is the
// struct of optional parameters for this action.
func (c *Client) SlugCreate(appIdentity string, processTypes map[string]string, options *SlugCreateOpts) (*Slug, error) {
	params := struct {
		ProcessTypes                 map[string]string `json:"process_types"`
		BuildpackProvidedDescription *string           `json:"buildpack_provided_description,omitempty"`
		Commit                       *string           `json:"commit,omitempty"`
	}{
		ProcessTypes: processTypes,
	}
	if options != nil {
		params.BuildpackProvidedDescription = options.BuildpackProvidedDescription
		params.Commit = options.Commit
	}
	var slugRes Slug
	return &slugRes, c.Post(&slugRes, "/apps/"+appIdentity+"/slugs", params)
}

// SlugCreateOpts holds the optional parameters for SlugCreate
type SlugCreateOpts struct {
	// description from buildpack of slug
	BuildpackProvidedDescription *string `json:"buildpack_provided_description,omitempty"`
	// identification of the code with your version control system (eg: SHA of the git HEAD)
	Commit *string `json:"commit,omitempty"`
}
