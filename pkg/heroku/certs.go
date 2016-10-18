package heroku

func (c *Client) CertsAttach(appIdentity string, options *CertsAttachOpts) error {
	return c.Post(nil, "/apps/"+appIdentity+"/certs", options)
}

type CertsAttachOpts struct {
	Cert    *string `json:"cert,omitempty"`
	Process *string `json:"process,omitempty"`
}
