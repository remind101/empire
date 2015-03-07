// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// A collaborator represents an account that has been given access to an app on
// Heroku.
type Collaborator struct {
	// when collaborator was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of collaborator
	Id string `json:"id"`

	// when collaborator was updated
	UpdatedAt time.Time `json:"updated_at"`

	// identity of collaborated account
	User struct {
		Email string `json:"email"`
		Id    string `json:"id"`
	} `json:"user"`
}

// Create a new collaborator.
//
// appIdentity is the unique identifier of the Collaborator's App. user is the
// unique email address of account or unique identifier of an account. options
// is the struct of optional parameters for this action.
func (c *Client) CollaboratorCreate(appIdentity string, user string, options *CollaboratorCreateOpts) (*Collaborator, error) {
	params := struct {
		User   string `json:"user"`
		Silent *bool  `json:"silent,omitempty"`
	}{
		User: user,
	}
	if options != nil {
		params.Silent = options.Silent
	}
	var collaboratorRes Collaborator
	return &collaboratorRes, c.Post(&collaboratorRes, "/apps/"+appIdentity+"/collaborators", params)
}

// CollaboratorCreateOpts holds the optional parameters for CollaboratorCreate
type CollaboratorCreateOpts struct {
	// whether to suppress email invitation when creating collaborator
	Silent *bool `json:"silent,omitempty"`
}

// Delete an existing collaborator.
//
// appIdentity is the unique identifier of the Collaborator's App.
// collaboratorIdentity is the unique identifier of the Collaborator.
func (c *Client) CollaboratorDelete(appIdentity string, collaboratorIdentity string) error {
	return c.Delete("/apps/" + appIdentity + "/collaborators/" + collaboratorIdentity)
}

// Info for existing collaborator.
//
// appIdentity is the unique identifier of the Collaborator's App.
// collaboratorIdentity is the unique identifier of the Collaborator.
func (c *Client) CollaboratorInfo(appIdentity string, collaboratorIdentity string) (*Collaborator, error) {
	var collaborator Collaborator
	return &collaborator, c.Get(&collaborator, "/apps/"+appIdentity+"/collaborators/"+collaboratorIdentity)
}

// List existing collaborators.
//
// appIdentity is the unique identifier of the Collaborator's App. lr is an
// optional ListRange that sets the Range options for the paginated list of
// results.
func (c *Client) CollaboratorList(appIdentity string, lr *ListRange) ([]Collaborator, error) {
	req, err := c.NewRequest("GET", "/apps/"+appIdentity+"/collaborators", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var collaboratorsRes []Collaborator
	return collaboratorsRes, c.DoReq(req, &collaboratorsRes)
}
