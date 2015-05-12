package empire

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq/hstore"
	"golang.org/x/net/context"
)

// Config represents a collection of environment variables.
type Config struct {
	ID   string
	Vars Vars

	AppID string
	App   *App
}

// NewConfig initializes a new config based on the old config, with the new
// variables provided.
func NewConfig(old *Config, vars Vars) *Config {
	v := mergeVars(old.Vars, vars)

	return &Config{
		AppID: old.AppID,
		Vars:  v,
	}
}

// Variable represents the name of an environment variable.
type Variable string

// Vars represents a variable -> value mapping.
type Vars map[Variable]string

// Scan implements the sql.Scanner interface.
func (v *Vars) Scan(src interface{}) error {
	h := hstore.Hstore{}
	if err := h.Scan(src); err != nil {
		return err
	}

	vars := make(Vars)

	for k, v := range h.Map {
		vars[Variable(k)] = v.String
	}

	*v = vars

	return nil
}

// Value implements the driver.Value interface.
func (v Vars) Value() (driver.Value, error) {
	m := make(map[string]sql.NullString)

	for k, v := range v {
		m[string(k)] = sql.NullString{
			Valid:  true,
			String: v,
		}
	}

	h := hstore.Hstore{
		Map: m,
	}

	return h.Value()
}

func (s *store) ConfigsCreate(config *Config) (*Config, error) {
	return configsCreate(s.db, config)
}

func (s *store) ConfigsFind(scope func(*gorm.DB) *gorm.DB) (*Config, error) {
	var config Config
	if err := s.db.Scopes(scope).Order("created_at desc").First(&config).Error; err != nil {
		if err == gorm.RecordNotFound {
			return nil, nil
		}

		return nil, err
	}
	return &config, nil
}

// ConfigsCreate inserts a Config in the database.
func configsCreate(db *gorm.DB, config *Config) (*Config, error) {
	return config, db.Create(config).Error
}

// ConfigID returns a scope to find a config by id.
func ConfigID(id string) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("id = ?", id)
	}
}

// ConfigApp returns a scope to find a config by app.
func ConfigApp(app *App) func(*gorm.DB) *gorm.DB {
	return func(db *gorm.DB) *gorm.DB {
		return db.Where("app_id = ?", app.ID)
	}
}

type configsService struct {
	store    *store
	releases *releasesService
}

func (s *configsService) ConfigsApply(ctx context.Context, app *App, vars Vars) (*Config, error) {
	old, err := s.ConfigsCurrent(app)
	if err != nil {
		return nil, err
	}

	c, err := s.store.ConfigsCreate(NewConfig(old, vars))
	if err != nil {
		return c, err
	}

	release, err := s.store.ReleasesLast(app)
	if err != nil {
		return c, err
	}

	if release != nil {
		keys := make([]string, 0, len(vars))
		for k, _ := range vars {
			keys = append(keys, string(k))
		}

		desc := fmt.Sprintf("Set %s config vars", strings.Join(keys, ","))

		// Create new release based on new config and old slug
		_, err = s.releases.ReleasesCreate(ctx, &Release{
			App:         app,
			Config:      c,
			Slug:        release.Slug,
			Description: desc,
		})
		if err != nil {
			return c, err
		}
	}

	return c, nil
}

// Returns configs for latest release or the latest configs if there are no releases.
func (s *configsService) ConfigsCurrent(app *App) (*Config, error) {
	r, err := s.store.ReleasesLast(app)
	if err != nil {
		return nil, err
	}

	var c *Config

	if r != nil {
		c = r.Config
	} else {
		// It's possible to have config without releases, this handles that.
		c, err = s.store.ConfigsFind(ConfigApp(app))
		if err != nil {
			return nil, err
		}
	}

	if c != nil {
		return c, nil
	}

	return s.store.ConfigsCreate(&Config{
		App:  app,
		Vars: make(Vars),
	})
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
