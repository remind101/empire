package empire

import (
	"crypto/sha1"
	"fmt"
	"sort"

	"github.com/remind101/empire/stores"
)

// ConfigVersion represents a unique identifier for a Config version.
type ConfigVersion string

// Config represents a collection of environment variables.
type Config struct {
	Version ConfigVersion `json:"version"`
	App     *App          `json:"app"`
	Vars    Vars          `json:"vars"`
}

// NewConfig initializes a new config based on the old config, with the new
// variables provided.
func NewConfig(old *Config, vars Vars) *Config {
	v := mergeVars(old.Vars, vars)

	return &Config{
		Version: ConfigVersion(hash(v)),
		App:     old.App,
		Vars:    v,
	}
}

// Variable represents the name of an environment variable.
type Variable string

// Vars represents a variable -> value mapping.
type Vars map[Variable]string

// ConfigsRepository represents an interface for retrieving and storing Config's.
type ConfigsRepository interface {
	// Head returns the current Config for the app.
	Head(AppName) (*Config, error)

	// Version returns the specific version of a Config for an app.
	Version(AppName, ConfigVersion) (*Config, error)

	// Store stores the Config for the app.
	Push(*Config) (*Config, error)
}

func NewConfigsRepository() ConfigsRepository {
	return &configsRepository{
		s: stores.NewMemStore(),
	}
}

func NewEtcdConfigsRepository(ns string) (ConfigsRepository, error) {
	s, err := stores.NewEtcdStore(ns)
	if err != nil {
		return nil, err
	}
	return &configsRepository{
		s: s,
	}, nil
}

// configsRepository is an in memory implementation of the Repository.
type configsRepository struct {
	s stores.Store
}

// Head implements Repository Head.
func (r *configsRepository) Head(appName AppName) (*Config, error) {
	c := &Config{}

	if ok, err := r.s.Get(keyHead(appName), c); err != nil || !ok {
		return nil, err
	}

	return c, nil
}

// Version implements Repository Version.
func (r *configsRepository) Version(appName AppName, version ConfigVersion) (*Config, error) {
	c := &Config{}

	if ok, err := r.s.Get(keyVersion(appName, version), c); err != nil || !ok {
		return nil, err
	}

	return c, nil
}

// Push implements Repository Push.
func (r *configsRepository) Push(config *Config) (*Config, error) {
	if err := r.s.Set(keyVersion(config.App.Name, config.Version), config); err != nil {
		return config, err
	}

	if err := r.s.Set(keyHead(config.App.Name), config); err != nil {
		return config, err
	}

	return config, nil
}

func keyHead(appName AppName) string {
	return fmt.Sprintf("%s/head", appName)
}

func keyVersion(appName AppName, version ConfigVersion) string {
	return fmt.Sprintf("%s/%s", appName, version)
}

// mergeVars copies all of the vars from a, and merges b into them, returning a
// new Vars.
func mergeVars(old, new Vars) Vars {
	vars := make(Vars)

	for n, v := range old {
		vars[n] = v
	}

	for n, v := range new {
		if v != "" {
			vars[n] = v
		} else {
			delete(vars, n)
		}
	}

	return vars
}

// hash creates a sha1 hash of a set of Vars.
func hash(vars Vars) string {
	s := make(sort.StringSlice, len(vars))

	for n := range vars {
		s = append(s, string(n))
	}

	sort.Sort(s)

	v := ""

	for _, n := range s {
		v = v + n + "=" + vars[Variable(n)]
	}

	return fmt.Sprintf("%x", sha1.Sum([]byte(v)))
}

// ConfigsService represents a service for interacting with Configs.
type ConfigsService interface {
	// Apply applies the vars to the apps latest Config.
	Apply(*App, Vars) (*Config, error)

	// Returns the Head Config for an App.
	Head(*App) (*Config, error)
}

// configsService is a base implementation of the ConfigsService.
type configsService struct {
	Repository ConfigsRepository
}

// NewConfigsService returns a new Service instance.
func NewConfigsService(options Options) (ConfigsService, error) {
	return &configsService{
		Repository: NewConfigsRepository(),
	}, nil
}

// Apply merges the provided Vars into the latest Config and returns a new
// Config.
func (s *configsService) Apply(app *App, vars Vars) (*Config, error) {
	l, err := s.Repository.Head(app.Name)
	if err != nil {
		return nil, err
	}

	// If the app doesn't have a config, just build a new one.
	if l == nil {
		l = &Config{
			App: app,
		}
	}

	c := NewConfig(l, vars)

	return s.Repository.Push(c)
}

// Gets the config for an app. If the app doesn't have a config, it will create
// a new one.
func (s *configsService) Head(app *App) (*Config, error) {
	c, err := s.Repository.Head(app.Name)
	if err != nil {
		return nil, err
	}

	if c == nil {
		return s.Repository.Push(&Config{
			App:  app,
			Vars: make(Vars),
		})
	}

	return c, nil
}
