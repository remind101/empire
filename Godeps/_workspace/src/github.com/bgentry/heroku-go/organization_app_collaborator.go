// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// An organization collaborator represents an account that has been given access
// to an organization app on Heroku.
type OrganizationAppCollaborator struct {
	// when collaborator was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of collaborator
	Id string `json:"id"`

	// role in the organization
	Role string `json:"role"`

	// when collaborator was updated
	UpdatedAt time.Time `json:"updated_at"`

	// identity of collaborated account
	User struct {
		Email string `json:"email"`
		Id    string `json:"id"`
	} `json:"user"`
}

// Create a new collaborator on an organization app. Use this endpoint instead
// of the /apps/{app_id_or_name}/collaborator endpoint when you want the
// collaborator to be granted [privileges]
// (https://devcenter.heroku.com/articles/org-users-access#roles) according to
// their role in the organization.
//
// appIdentity is the unique identifier of the OrganizationAppCollaborator's
// App. user is the unique email address of account or unique identifier of an
// account. options is the struct of optional parameters for this action.
func (c *Client) OrganizationAppCollaboratorCreate(appIdentity string, user string, options *OrganizationAppCollaboratorCreateOpts) (*OrganizationAppCollaborator, error) {
	params := struct {
		User   string `json:"user"`
		Silent *bool  `json:"silent,omitempty"`
	}{
		User: user,
	}
	if options != nil {
		params.Silent = options.Silent
	}
	var organizationAppCollaboratorRes OrganizationAppCollaborator
	return &organizationAppCollaboratorRes, c.Post(&organizationAppCollaboratorRes, "/organizations/apps/"+appIdentity+"/collaborators", params)
}

// OrganizationAppCollaboratorCreateOpts holds the optional parameters for OrganizationAppCollaboratorCreate
type OrganizationAppCollaboratorCreateOpts struct {
	// whether to suppress email invitation when creating collaborator
	Silent *bool `json:"silent,omitempty"`
}

// Delete an existing collaborator from an organization app.
//
// appIdentity is the unique identifier of the OrganizationAppCollaborator's
// App. collaboratorIdentity is the unique identifier of the
// OrganizationAppCollaborator's Collaborator.
func (c *Client) OrganizationAppCollaboratorDelete(appIdentity string, collaboratorIdentity string) error {
	return c.Delete("/organizations/apps/" + appIdentity + "/collaborators/" + collaboratorIdentity)
}

// Info for a collaborator on an organization app.
//
// appIdentity is the unique identifier of the OrganizationAppCollaborator's
// App. collaboratorIdentity is the unique identifier of the
// OrganizationAppCollaborator's Collaborator.
func (c *Client) OrganizationAppCollaboratorInfo(appIdentity string, collaboratorIdentity string) (*OrganizationAppCollaborator, error) {
	var organizationAppCollaborator OrganizationAppCollaborator
	return &organizationAppCollaborator, c.Get(&organizationAppCollaborator, "/organizations/apps/"+appIdentity+"/collaborators/"+collaboratorIdentity)
}

// List collaborators on an organization app.
//
// appIdentity is the unique identifier of the OrganizationAppCollaborator's
// App. lr is an optional ListRange that sets the Range options for the
// paginated list of results.
func (c *Client) OrganizationAppCollaboratorList(appIdentity string, lr *ListRange) ([]OrganizationAppCollaborator, error) {
	req, err := c.NewRequest("GET", "/organizations/apps/"+appIdentity+"/collaborators", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var organizationAppCollaboratorsRes []OrganizationAppCollaborator
	return organizationAppCollaboratorsRes, c.DoReq(req, &organizationAppCollaboratorsRes)
}
