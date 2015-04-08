// package plugin is a small framework for building Go binaries that contain
// plugins for the heroku hk command https://github.com/heroku/hk.
package plugin

// A plugin represents an individual plugin.
type Plugin struct {
	// The name of the plugin.
	Name string

	// The action that will be performed when this plugin is invoked.
	Action func(*Context)
}
