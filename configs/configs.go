package configs

import (
	"crypto/sha1"
	"fmt"
	"sort"

	"github.com/remind101/empire/apps"
)

// Version represents a unique identifier for a Config version.
type Version string

// Config represents a collection of environment variables.
type Config struct {
	Version Version
	App     *apps.App
	Vars    Vars
}

// Variable represents the name of an environment variable.
type Variable string

// Vars represents a variable -> value mapping.
type Vars map[Variable]string

// Repository represents an interface for retrieving and storing Config's.
type Repository interface {
	// Head returns the current Config for the app.
	Head(apps.ID) (*Config, error)

	// Version returns the specific version of a Config for an app.
	Version(apps.ID, Version) (*Config, error)

	// Store stores the Config for the app.
	Push(*Config) (*Config, error)
}

// repository is an in memory implementation of the Repository.
type repository struct {
	// Maps an app to an array of Config objects.
	s map[apps.ID][]*Config

	// Keeps a reference to the current Config object for the app.
	h map[apps.ID]*Config
}

func newRepository() *repository {
	return &repository{
		s: make(map[apps.ID][]*Config),
		h: make(map[apps.ID]*Config),
	}
}

// Head implements Repository Head.
func (r *repository) Head(appID apps.ID) (*Config, error) {
	if r.h[appID] == nil {
		return nil, nil
	}

	return r.h[appID], nil
}

// Version implements Repository Version.
func (r *repository) Version(appID apps.ID, version Version) (*Config, error) {
	for _, c := range r.s[appID] {
		if c.Version == version {
			return c, nil
		}
	}

	return nil, nil
}

// Push implements Repository Push.
func (r *repository) Push(config *Config) (*Config, error) {
	id := config.App.ID

	r.s[id] = append(r.s[id], config)
	r.h[id] = config

	return config, nil
}

// Service represents a service for manipulating the Config for a repo.
type Service struct {
	Repository
}

// NewService returns a new Service instance.
func NewService(r Repository) *Service {
	if r == nil {
		r = newRepository()
	}

	return &Service{Repository: r}
}

// Apply merges the provided Vars into the latest Config and returns a new
// Config.
func (s *Service) Apply(app *apps.App, vars Vars) (*Config, error) {
	l, err := s.Repository.Head(app.ID)

	if err != nil {
		return nil, err
	}

	if l == nil {
		l = &Config{
			App: app,
		}
	}

	c := newConfig(l, vars)

	return s.Repository.Push(c)
}

// newConfig creates a new config based on the old config, with the new
// variables provided.
func newConfig(config *Config, vars Vars) *Config {
	v := mergeVars(config.Vars, vars)

	return &Config{
		Version: Version(hash(v)),
		App:     config.App,
		Vars:    v,
	}
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
