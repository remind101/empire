// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// Dynos encapsulate running processes of an app on Heroku.
type Dyno struct {
	// a URL to stream output from for attached processes or null for non-attached processes
	AttachURL *string `json:"attach_url"`

	// command used to start this process
	Command string `json:"command"`

	// when dyno was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of this dyno
	Id string `json:"id"`

	// the name of this process on this dyno
	Name string `json:"name"`

	// app release of the dyno
	Release struct {
		Id      string `json:"id"`
		Version int    `json:"version"`
	} `json:"release"`

	// dyno size (default: "1X")
	Size string `json:"size"`

	// current status of process (either: crashed, down, idle, starting, or up)
	State string `json:"state"`

	// type of process
	Type string `json:"type"`

	// when process last changed state
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new dyno.
//
// appIdentity is the unique identifier of the Dyno's App. command is the
// command used to start this process. options is the struct of optional
// parameters for this action.
func (c *Client) DynoCreate(appIdentity string, command string, options *DynoCreateOpts) (*Dyno, error) {
	params := struct {
		Command string             `json:"command"`
		Attach  *bool              `json:"attach,omitempty"`
		Env     *map[string]string `json:"env,omitempty"`
		Size    *string            `json:"size,omitempty"`
	}{
		Command: command,
	}
	if options != nil {
		params.Attach = options.Attach
		params.Env = options.Env
		params.Size = options.Size
	}
	var dynoRes Dyno
	return &dynoRes, c.Post(&dynoRes, "/apps/"+appIdentity+"/dynos", params)
}

// DynoCreateOpts holds the optional parameters for DynoCreate
type DynoCreateOpts struct {
	// whether to stream output or not
	Attach *bool `json:"attach,omitempty"`
	// custom environment to add to the dyno config vars
	Env *map[string]string `json:"env,omitempty"`
	// dyno size (default: "1X")
	Size *string `json:"size,omitempty"`
}

// Restart dyno.
//
// appIdentity is the unique identifier of the Dyno's App. dynoIdentity is the
// unique identifier of the Dyno.
func (c *Client) DynoRestart(appIdentity string, dynoIdentity string) error {
	return c.Delete("/apps/" + appIdentity + "/dynos/" + dynoIdentity)
}

// Restart all dynos
//
// appIdentity is the unique identifier of the Dyno's App.
func (c *Client) DynoRestartAll(appIdentity string) error {
	return c.Delete("/apps/" + appIdentity + "/dynos")
}

// Info for existing dyno.
//
// appIdentity is the unique identifier of the Dyno's App. dynoIdentity is the
// unique identifier of the Dyno.
func (c *Client) DynoInfo(appIdentity string, dynoIdentity string) (*Dyno, error) {
	var dyno Dyno
	return &dyno, c.Get(&dyno, "/apps/"+appIdentity+"/dynos/"+dynoIdentity)
}

// List existing dynos.
//
// appIdentity is the unique identifier of the Dyno's App. lr is an optional
// ListRange that sets the Range options for the paginated list of results.
func (c *Client) DynoList(appIdentity string, lr *ListRange) ([]Dyno, error) {
	req, err := c.NewRequest("GET", "/apps/"+appIdentity+"/dynos", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var dynosRes []Dyno
	return dynosRes, c.DoReq(req, &dynosRes)
}
