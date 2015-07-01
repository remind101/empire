// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

// Organizations allow you to manage access to a shared group of applications
// across your development team.
type Organization struct {
	// whether charges incurred by the org are paid by credit card.
	CreditCardCollections bool `json:"credit_card_collections"`

	// whether to use this organization when none is specified
	Default bool `json:"default"`

	// unique name of organization
	Name string `json:"name"`

	// whether the org is provisioned licenses by salesforce.
	ProvisionedLicenses bool `json:"provisioned_licenses"`

	// role in the organization
	Role string `json:"role"`
}

// List organizations in which you are a member.
//
// lr is an optional ListRange that sets the Range options for the paginated
// list of results.
func (c *Client) OrganizationList(lr *ListRange) ([]Organization, error) {
	req, err := c.NewRequest("GET", "/organizations", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var organizationsRes []Organization
	return organizationsRes, c.DoReq(req, &organizationsRes)
}

// Set or Unset the organization as your default organization.
//
// organizationIdentity is the unique identifier of the Organization. options is
// the struct of optional parameters for this action.
func (c *Client) OrganizationUpdate(organizationIdentity string, options *OrganizationUpdateOpts) (*Organization, error) {
	var organizationRes Organization
	return &organizationRes, c.Patch(&organizationRes, "/organizations/"+organizationIdentity, options)
}

// OrganizationUpdateOpts holds the optional parameters for OrganizationUpdate
type OrganizationUpdateOpts struct {
	// whether to use this organization when none is specified
	Default *bool `json:"default,omitempty"`
}
