package empire

import (
	"database/sql"
	"database/sql/driver"
	"fmt"
	"sort"
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
type Vars map[Variable]*string

// Scan implements the sql.Scanner interface.
func (v *Vars) Scan(src interface{}) error {
	h := hstore.Hstore{}
	if err := h.Scan(src); err != nil {
		return err
	}

	vars := make(Vars)

	for k, v := range h.Map {
		// Go reuses the same address space for v, so &v.String would always
		// return the same address
		tmp := v.String
		vars[Variable(k)] = &tmp
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
			String: *v,
		}
	}

	h := hstore.Hstore{
		Map: m,
	}

	return h.Value()
}

// ConfigsQuery is a Scope implementation for common things to filter releases
// by.
type ConfigsQuery struct {
	// If provided, returns finds the config with the given id.
	ID *string

	// If provided, filters configs for the given app.
	App *App
}

// Scope implements the Scope interface.
func (q ConfigsQuery) Scope(db *gorm.DB) *gorm.DB {
	var scope ComposedScope

	if q.ID != nil {
		scope = append(scope, ID(*q.ID))
	}

	if q.App != nil {
		scope = append(scope, ForApp(q.App))
	}

	return scope.Scope(db)
}

// ConfigsFirst returns the first matching config.
func (s *store) ConfigsFirst(scope Scope) (*Config, error) {
	var config Config
	scope = ComposedScope{Order("created_at desc"), scope}
	return &config, s.First(scope, &config)
}

// ConfigsCreate persists the Config.
func (s *store) ConfigsCreate(config *Config) (*Config, error) {
	return configsCreate(s.db, config)
}

// ConfigsCreate inserts a Config in the database.
func configsCreate(db *gorm.DB, config *Config) (*Config, error) {
	return config, db.Create(config).Error
}

type configsService struct {
	*Empire
}

func (s *configsService) Set(ctx context.Context, opts SetOpts) (*Config, error) {
	app, vars := opts.App, opts.Vars

	old, err := s.Config(app)
	if err != nil {
		return nil, err
	}

	c, err := s.store.ConfigsCreate(NewConfig(old, vars))
	if err != nil {
		return c, err
	}

	release, err := s.store.ReleasesFind(ReleasesQuery{App: app})
	if err != nil {
		if err == gorm.RecordNotFound {
			err = nil
		}

		return c, err
	}

	// Create new release based on new config and old slug
	_, err = s.releases.ReleasesCreate(ctx, &Release{
		App:         release.App,
		Config:      c,
		Slug:        release.Slug,
		Description: configsApplyReleaseDesc(vars),
	})
	return c, err
}

// Returns configs for latest release or the latest configs if there are no releases.
func (s *configsService) Config(app *App) (*Config, error) {
	r, err := s.store.ReleasesFind(ReleasesQuery{App: app})
	if err != nil {
		if err == gorm.RecordNotFound {
			// It's possible to have config without releases, this handles that.
			c, err := s.store.ConfigsFirst(ConfigsQuery{App: app})
			if err != nil {
				if err == gorm.RecordNotFound {
					return s.store.ConfigsCreate(&Config{
						App:  app,
						Vars: make(Vars),
					})
				}
				return nil, err
			}

			return c, nil
		}

		return nil, err
	}

	return r.Config, nil
}

// mergeVars copies all of the vars from a, and merges b into them, returning a
// new Vars.
func mergeVars(old, new Vars) Vars {
	vars := make(Vars)

	for n, v := range old {
		vars[n] = v
	}

	for n, v := range new {
		if v == nil {
			delete(vars, n)
		} else {
			vars[n] = v
		}
	}

	return vars
}

// configsApplyReleaseDesc formats a release description based on the config variables
// being applied.
func configsApplyReleaseDesc(vars Vars) string {
	verb := "Set"
	plural := ""
	if len(vars) > 1 {
		plural = "s"
	}

	keys := make(sort.StringSlice, 0, len(vars))
	for k, v := range vars {
		keys = append(keys, string(k))
		if v == nil {
			verb = "Unset"
		}
	}
	keys.Sort()
	return fmt.Sprintf("%s %s config var%s", verb, strings.Join(keys, ", "), plural)
}
