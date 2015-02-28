// WARNING: This code is auto-generated from the Heroku Platform API JSON Schema
// by a Ruby script (gen/gen.rb). Changes should be made to the generation
// script rather than the generated files.

package heroku

import (
	"time"
)

// A log session is a reference to the http based log stream for an app.
type LogSession struct {
	// when log connection was created
	CreatedAt time.Time `json:"created_at"`

	// unique identifier of this log session
	Id string `json:"id"`

	// URL for log streaming session
	LogplexURL string `json:"logplex_url"`

	// when log session was updated
	UpdatedAt time.Time `json:"updated_at"`
}

// Create a new log session.
//
// appIdentity is the unique identifier of the LogSession's App. options is the
// struct of optional parameters for this action.
func (c *Client) LogSessionCreate(appIdentity string, options *LogSessionCreateOpts) (*LogSession, error) {
	var logSessionRes LogSession
	return &logSessionRes, c.Post(&logSessionRes, "/apps/"+appIdentity+"/log-sessions", options)
}

// LogSessionCreateOpts holds the optional parameters for LogSessionCreate
type LogSessionCreateOpts struct {
	// dyno to limit results to
	Dyno *string `json:"dyno,omitempty"`
	// number of log lines to stream at once
	Lines *int `json:"lines,omitempty"`
	// log source to limit results to
	Source *string `json:"source,omitempty"`
	// whether to stream ongoing logs
	Tail *bool `json:"tail,omitempty"`
}
