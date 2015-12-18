// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// Log drains provide a way to forward your Heroku logs to an external syslog
// server for long-term archiving. This external service must be configured to
// receive syslog packets from Heroku, whereupon its URL can be added to an app
// using this API. Some addons will add a log drain when they are provisioned to
// an app. These drains can only be removed by removing the add-on.
type LogDrain struct {
	// addon that created the drain
	Addon *struct {
		Id string `json:"id"`
	} `json:"addon"`

	// when log drain was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of this log drain
	Id string `json:"id"`

	// token associated with the log drain
	Token string `json:"token"`

	// when log drain was updated
	UpdatedAt time.Time `json:"updated_at"`

	// url associated with the log drain
	URL string `json:"url"`
}

// Create a new log drain.
//
// appIdentity is the unique identifier of the LogDrain's App. url is the url
// associated with the log drain.
func (c *Client) LogDrainCreate(appIdentity string, url string) (*LogDrain, error) {
	params := struct {
		URL string `json:"url"`
	}{
		URL: url,
	}
	var logDrainRes LogDrain
	return &logDrainRes, c.Post(&logDrainRes, "/apps/"+appIdentity+"/log-drains", params)
}

// Delete an existing log drain. Log drains added by add-ons can only be removed
// by removing the add-on.
//
// appIdentity is the unique identifier of the LogDrain's App. logDrainIdentity
// is the unique identifier of the LogDrain.
func (c *Client) LogDrainDelete(appIdentity string, logDrainIdentity string) error {
	return c.Delete("/apps/" + appIdentity + "/log-drains/" + logDrainIdentity)
}

// Info for existing log drain.
//
// appIdentity is the unique identifier of the LogDrain's App. logDrainIdentity
// is the unique identifier of the LogDrain.
func (c *Client) LogDrainInfo(appIdentity string, logDrainIdentity string) (*LogDrain, error) {
	var logDrain LogDrain
	return &logDrain, c.Get(&logDrain, "/apps/"+appIdentity+"/log-drains/"+logDrainIdentity)
}

// List existing log drains.
//
// appIdentity is the unique identifier of the LogDrain's App. lr is an optional
// ListRange that sets the Range options for the paginated list of results.
func (c *Client) LogDrainList(appIdentity string, lr *ListRange) ([]LogDrain, error) {
	req, err := c.NewRequest("GET", "/apps/"+appIdentity+"/log-drains", nil)
	if err != nil {
		return nil, err
	}

	if lr != nil {
		lr.SetHeader(req)
	}

	var logDrainsRes []LogDrain
	return logDrainsRes, c.DoReq(req, &logDrainsRes)
}
