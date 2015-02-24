package configs

import (
	"crypto/sha1"
	"fmt"
	"sort"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/stores"
)

// Version represents a unique identifier for a Config version.
type Version string

// Config represents a collection of environment variables.
type Config struct {
	Version Version   `json:"version"`
	App     *apps.App `json:"app"`
	Vars    Vars      `json:"vars"`
}

// NewConfig initializes a new config based on the old config, with the new
// variables provided.
func NewConfig(old *Config, vars Vars) *Config {
	v := mergeVars(old.Vars, vars)

	return &Config{
		Version: Version(hash(v)),
		App:     old.App,
		Vars:    v,
	}
}

// Variable represents the name of an environment variable.
type Variable string

// Vars represents a variable -> value mapping.
type Vars map[Variable]string

// Repository represents an interface for retrieving and storing Config's.
type Repository interface {
	// Head returns the current Config for the app.
	Head(apps.Name) (*Config, error)

	// Version returns the specific version of a Config for an app.
	Version(apps.Name, Version) (*Config, error)

	// Store stores the Config for the app.
	Push(*Config) (*Config, error)
}

type repository struct {
	s stores.Store
}

func NewRepository() Repository {
	return &repository{stores.NewMemStore()}
}

func NewEtcdRepository(ns string) (Repository, error) {
	s, err := stores.NewEtcdStore(ns)
	if err != nil {
		return nil, err
	}
	return &repository{s}, nil
}

// Head implements Repository Head.
func (r *repository) Head(appName apps.Name) (*Config, error) {
	c := &Config{}

	if ok, err := r.s.Get(keyHead(appName), c); err != nil || !ok {
		return nil, err
	}

	return c, nil
}

// Version implements Repository Version.
func (r *repository) Version(appName apps.Name, version Version) (*Config, error) {
	c := &Config{}

	if ok, err := r.s.Get(keyVersion(appName, version), c); err != nil || !ok {
		return nil, err
	}

	return c, nil
}

// Push implements Repository Push.
func (r *repository) Push(config *Config) (*Config, error) {
	if err := r.s.Set(keyVersion(config.App.Name, config.Version), config); err != nil {
		return config, err
	}

	if err := r.s.Set(keyHead(config.App.Name), config); err != nil {
		return config, err
	}

	return config, nil
}

func keyHead(appName apps.Name) string {
	return fmt.Sprintf("%s/head", appName)
}

func keyVersion(appName apps.Name, version Version) string {
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
