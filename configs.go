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
	// A unique uuid representing this Config.
	ID string

	// The environment variables in this config.
	Vars Vars

	// The id of the app that this config relates to.
	AppID string

	// The app that this config relates to.
	App *App
}

// newConfig initializes a new config based on the old config, with the new
// variables provided.
func newConfig(old *Config, vars Vars) *Config {
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

// ConfigsQuery is a scope implementation for common things to filter releases
// by.
type ConfigsQuery struct {
	// If provided, returns finds the config with the given id.
	ID *string

	// If provided, filters configs for the given app.
	App *App
}

// scope implements the scope interface.
func (q ConfigsQuery) scope(db *gorm.DB) *gorm.DB {
	var scope composedScope

	if q.ID != nil {
		scope = append(scope, idEquals(*q.ID))
	}

	if q.App != nil {
		scope = append(scope, forApp(q.App))
	}

	return scope.scope(db)
}

// configsFind returns the first matching config.
func configsFind(db *gorm.DB, scope scope) (*Config, error) {
	var config Config
	scope = composedScope{order("created_at desc"), scope}
	return &config, first(db, scope, &config)
}

// ConfigsCreate inserts a Config in the database.
func configsCreate(db *gorm.DB, config *Config) (*Config, error) {
	return config, db.Create(config).Error
}

type configsService struct {
	*Empire
}

func (s *configsService) Set(ctx context.Context, db *gorm.DB, opts SetOpts) (*Config, *Config, error) {
	app, vars := opts.App, opts.Vars

	old, err := s.Config(db, app)
	if err != nil {
		return nil, old, err
	}

	c, err := configsCreate(db, newConfig(old, vars))
	if err != nil {
		return c, old, err
	}

	release, err := releasesFind(db, ReleasesQuery{App: app})
	if err != nil {
		if err == gorm.RecordNotFound {
			err = nil
		}

		return c, old, err
	}

	// Create new release based on new config and old slug
	_, err = s.releases.CreateAndRelease(ctx, db, &Release{
		App:         release.App,
		Config:      c,
		Slug:        release.Slug,
		Description: configsApplyReleaseDesc(DiffVars(c.Vars, old.Vars), opts),
	}, nil)
	return c, old, err
}

// Returns configs for latest release or the latest configs if there are no releases.
func (s *configsService) Config(db *gorm.DB, app *App) (*Config, error) {
	r, err := releasesFind(db, ReleasesQuery{App: app})
	if err != nil {
		if err == gorm.RecordNotFound {
			// It's possible to have config without releases, this handles that.
			c, err := configsFind(db, ConfigsQuery{App: app})
			if err != nil {
				if err == gorm.RecordNotFound {
					// Return an empty config.
					return &Config{
						AppID: app.ID,
						App:   app,
						Vars:  make(Vars),
					}, nil
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

// VarsDiff can be used to compare what's different between two configs.
type VarsDiff struct {
	// Environment variables that were previously set, but changed to a new value
	Changed []string

	// Environment variables that were previously set, but removed.
	Removed []string

	// Environment variables that were previously unset, but were added.
	Added []string
}

func (d *VarsDiff) String() string {
	totalChanges := len(d.Changed) + len(d.Added) + len(d.Removed)
	if totalChanges == 0 {
		return "Made no changes to config vars"
	}

	plural := ""
	if totalChanges > 1 {
		plural = "s"
	}

	var parts []string

	if v := d.Added; len(v) > 0 {
		parts = append(parts, fmt.Sprintf("Added (%s)", strings.Join(v, ", ")))
	}
	if v := d.Changed; len(v) > 0 {
		parts = append(parts, fmt.Sprintf("Changed (%s)", strings.Join(v, ", ")))
	}
	if v := d.Removed; len(v) > 0 {
		parts = append(parts, fmt.Sprintf("Removed (%s)", strings.Join(v, ", ")))
	}

	return fmt.Sprintf("%s config var%s", strings.Join(parts, " "), plural)
}

// DiffVars generates a diff between two Vars objects, treating `a` as the newer
// version. It can tell you what was added, changed or removed between the two
// objects.
func DiffVars(a, b Vars) *VarsDiff {
	var added, changed, removed sort.StringSlice
	for k, va := range a {
		// Key didn't exist in the old config, but does in the new one.
		if vb, ok := b[k]; !ok {
			added = append(added, string(k))
		} else {
			// Key exists in both configs, let's check if the value
			// has changed.
			if *va != *vb {
				changed = append(changed, string(k))
			}
		}
	}
	for k := range b {
		// Key doesn't exist in the new config, but does in the old one.
		if _, ok := a[k]; !ok {
			removed = append(removed, string(k))
		}
	}
	added.Sort()
	changed.Sort()
	removed.Sort()
	return &VarsDiff{
		Added:   added,
		Changed: changed,
		Removed: removed,
	}
}

// configsApplyReleaseDesc formats a release description based on the config variables
// being applied.
func configsApplyReleaseDesc(diff *VarsDiff, opts SetOpts) string {
	return appendMessageToDescription(diff.String(), opts.User, opts.Message)
}
