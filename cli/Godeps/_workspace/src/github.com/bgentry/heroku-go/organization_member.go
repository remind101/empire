// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// An organization member is an individual with access to an organization.
type OrganizationMember struct {
	// when organization-member was created
	CreatedAt time.Time `json:"created_at"`

	// email address of the organization member
	Email string `json:"email"`

	// role in the organization
	Role string `json:"role"`

	// when organization-member was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new organization member, or update their role.
//
// organizationIdentity is the unique identifier of the OrganizationMember's
// Organization. email is the email address of the organization member. role is
// the role in the organization.
func (c *Client) OrganizationMemberCreateOrUpdate(organizationIdentity string, email string, role string) (*OrganizationMember, error) {
	params := struct {
		Email string `json:"email"`
		Role  string `json:"role"`
	}{
		Email: email,
		Role:  role,
	}
	var organizationMemberRes OrganizationMember
	return &organizationMemberRes, c.Put(&organizationMemberRes, "/organizations/"+organizationIdentity+"/members", params)
}

// Remove a member from the organization.
//
// organizationIdentity is the unique identifier of the OrganizationMember's
// Organization. organizationMemberIdentity is the unique identifier of the
// OrganizationMember.
func (c *Client) OrganizationMemberDelete(organizationIdentity string, organizationMemberIdentity string) error {
	return c.Delete("/organizations/" + organizationIdentity + "/members/" + organizationIdentity)
}

// List members of the organization.
//
// organizationIdentity is the unique identifier of the OrganizationMember's
// Organization. lr is an optional ListRange that sets the Range options for the
// paginated list of results.
func (c *Client) OrganizationMemberList(organizationIdentity string, lr *ListRange) ([]OrganizationMember, error) {
	req, err := c.NewRequest("GET", "/organizations/"+organizationIdentity+"/members", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var organizationMembersRes []OrganizationMember
	return organizationMembersRes, c.DoReq(req, &organizationMembersRes)
}
