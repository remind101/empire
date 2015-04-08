package plugin

import (
	"os"

	"github.com/bgentry/heroku-go"
)

// Context is provided to plugins when they are run.
type Context struct {
	// The name of the app that this plugin should be invoked against. The
	// zero value means that no app was provided.
	App string

	// A pre-configured heroku client.
	Client *heroku.Client
}

func NewContext() *Context {
	return &Context{
		App:    os.Getenv("HKAPP"),
		Client: NewClient(),
	}
}

// NewClient returns a new heroku client configured from the environment
// variables that the hk client sets.
func NewClient() *heroku.Client {
	c := &heroku.Client{
		Username: os.Getenv("HKUSER"),
		Password: os.Getenv("HKPASS"),
	}
	c.URL = os.Getenv("HEROKU_API_URL")
	return c
}
