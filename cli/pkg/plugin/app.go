package plugin

import (
	"errors"
	"fmt"
)

// An App represents a collection of plugins that this binary exposes.
type App struct {
	Plugins []Plugin
}

func NewApp() *App {
	return &App{}
}

// Run runs the proper plugin.
func (a *App) Run(arguments []string) error {
	if len(arguments) == 0 {
		return errors.New("no plugin name provided")
	}

	name := arguments[0]
	plugin := findPlugin(name, a.Plugins)

	if plugin == nil {
		return fmt.Errorf("plugin %s not found", name)
	}

	plugin.Action(NewContext(arguments))

	return nil
}

func findPlugin(name string, plugins []Plugin) *Plugin {
	for _, p := range plugins {
		if p.Name == name {
			return &p
		}
	}

	return nil
}
