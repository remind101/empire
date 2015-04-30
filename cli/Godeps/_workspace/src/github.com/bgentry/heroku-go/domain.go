// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// Domains define what web routes should be routed to an app on Heroku.
type Domain struct {
	// when domain was created
	CreatedAt time.Time `json:"created_at"`

	// full hostname
	Hostname string `json:"hostname"`

	// unique identifier of this domain
	Id string `json:"id"`

	// when domain was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new domain.
//
// appIdentity is the unique identifier of the Domain's App. hostname is the
// full hostname.
func (c *Client) DomainCreate(appIdentity string, hostname string) (*Domain, error) {
	params := struct {
		Hostname string `json:"hostname"`
	}{
		Hostname: hostname,
	}
	var domainRes Domain
	return &domainRes, c.Post(&domainRes, "/apps/"+appIdentity+"/domains", params)
}

// Delete an existing domain
//
// appIdentity is the unique identifier of the Domain's App. domainIdentity is
// the unique identifier of the Domain.
func (c *Client) DomainDelete(appIdentity string, domainIdentity string) error {
	return c.Delete("/apps/" + appIdentity + "/domains/" + domainIdentity)
}

// Info for existing domain.
//
// appIdentity is the unique identifier of the Domain's App. domainIdentity is
// the unique identifier of the Domain.
func (c *Client) DomainInfo(appIdentity string, domainIdentity string) (*Domain, error) {
	var domain Domain
	return &domain, c.Get(&domain, "/apps/"+appIdentity+"/domains/"+domainIdentity)
}

// List existing domains.
//
// appIdentity is the unique identifier of the Domain's App. lr is an optional
// ListRange that sets the Range options for the paginated list of results.
func (c *Client) DomainList(appIdentity string, lr *ListRange) ([]Domain, error) {
	req, err := c.NewRequest("GET", "/apps/"+appIdentity+"/domains", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var domainsRes []Domain
	return domainsRes, c.DoReq(req, &domainsRes)
}
