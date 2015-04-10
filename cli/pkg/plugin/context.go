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

	// The command-line arguments, minus the program name and plugin name.
	Args []string
}

func NewContext(args []string) *Context {
	return &Context{
		App:    os.Getenv("HKAPP"),
		Client: NewClient(),
		Args:   args[1:],
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
