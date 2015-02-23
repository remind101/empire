package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
)

// ConfigsService represents a service for interacting with Configs.
type ConfigsService interface {
	// Apply applies the vars to the apps latest Config.
	Apply(*apps.App, configs.Vars) (*configs.Config, error)

	// Returns the Head Config for an App.
	Head(*apps.App) (*configs.Config, error)
}

// configsService is a base implementation of the ConfigsService.
type configsService struct {
	configs.Repository
}

// NewConfigsService returns a new Service instance.
func NewConfigsService(options Options) (ConfigsService, error) {
	return &configsService{
		Repository: configs.NewRepository(),
	}, nil
}

// Apply merges the provided Vars into the latest Config and returns a new
// Config.
func (s *configsService) Apply(app *apps.App, vars configs.Vars) (*configs.Config, error) {
	l, err := s.Repository.Head(app.Name)
	if err != nil {
		return nil, err
	}

	// If the app doesn't have a config, just build a new one.
	if l == nil {
		l = &configs.Config{
			App: app,
		}
	}

	c := configs.NewConfig(l, vars)

	return s.Repository.Push(c)
}

// Gets the config for an app. If the app doesn't have a config, it will create
// a new one.
func (s *configsService) Head(app *apps.App) (*configs.Config, error) {
	c, err := s.Repository.Head(app.Name)
	if err != nil {
		return nil, err
	}

	if c == nil {
		return s.Repository.Push(&configs.Config{
			App:  app,
			Vars: make(configs.Vars),
		})
	}

	return c, nil
}
