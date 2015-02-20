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

// configsService is an implementation of the ConfigsService.
type configsService struct {
	configs.Repository
}

// NewConfigsService returns a new Service instance.
func NewConfigsService(r configs.Repository) ConfigsService {
	if r == nil {
		r = configs.NewRepository()
	}

	return &configsService{
		Repository: r,
	}
}

// Apply merges the provided Vars into the latest Config and returns a new
// Config.
func (s *configsService) Apply(app *apps.App, vars configs.Vars) (*configs.Config, error) {
	l, err := s.Repository.Head(app.Name)

	if err != nil {
		return nil, err
	}

	if l == nil {
		l = &configs.Config{
			App: app,
		}
	}

	c := configs.NewConfig(l, vars)

	return s.Repository.Push(c)
}

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
