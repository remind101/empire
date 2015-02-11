package configs

import (
	"crypto/sha1"
	"fmt"
	"sort"

	"github.com/remind101/empire/repos"
)

// Config represents a collection of environment variables.
type Config struct {
	Version string
	Repo    repos.Repo
	Vars    Vars
}

// Variable represents the name of an environment variable.
type Variable string

// Vars represents a variable -> value mapping.
type Vars map[Variable]string

// ConfigRepository represents an interface for retrieving and storing Config's.
type ConfigRepository interface {
	// Head returns the current Config for the app.
	Head(repos.Repo) (*Config, error)

	// Version returns the specific version of a Config for an app.
	Version(repos.Repo, string) (*Config, error)

	// Store stores the Config for the app.
	Push(repos.Repo, *Config) (*Config, error)
}

// configRepository is an in memory implementation of the ConfigRepository.
type configRepository struct {
	// Maps an app to an array of Config objects.
	s map[repos.Repo][]*Config

	// Keeps a reference to the current Config object for the app.
	h map[repos.Repo]*Config
}

func newConfigRepository() *configRepository {
	return &configRepository{
		s: make(map[repos.Repo][]*Config),
		h: make(map[repos.Repo]*Config),
	}
}

// Head implements ConfigRepository Head.
func (r *configRepository) Head(repo repos.Repo) (*Config, error) {
	if r.h[repo] == nil {
		return nil, nil
	}

	return r.h[repo], nil
}

// Version implements ConfigRepository Version.
func (r *configRepository) Version(repo repos.Repo, version string) (*Config, error) {
	for _, c := range r.s[repo] {
		if c.Version == version {
			return c, nil
		}
	}

	return nil, nil
}

// Push implements ConfigRepository Push.
func (r *configRepository) Push(repo repos.Repo, config *Config) (*Config, error) {
	r.s[repo] = append(r.s[repo], config)
	r.h[repo] = config

	return config, nil
}

// ConfigService represents a service for manipulating the Config for a repo.
type ConfigService struct {
	ConfigRepository
}

// Apply merges the provided Vars into the latest Config and returns a new
// Config.
func (s *ConfigService) Apply(repo repos.Repo, vars Vars) (*Config, error) {
	l, err := s.ConfigRepository.Head(repo)

	if err != nil {
		return nil, err
	}

	if l == nil {
		l = &Config{
			Repo: repo,
		}
	}

	c := newConfig(l, vars)

	return s.ConfigRepository.Push(repo, c)
}

// newConfig creates a new config based on the old config, with the new
// variables provided.
func newConfig(config *Config, vars Vars) *Config {
	v := mergeVars(config.Vars, vars)

	return &Config{
		Version: hash(v),
		Repo:    config.Repo,
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
