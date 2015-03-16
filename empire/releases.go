package empire

import (
	"database/sql"
	"time"

	"gopkg.in/gorp.v1"
)

// Release is a combination of a Config and a Slug, which form a deployable
// release.
type Release struct {
	ID  string `json:"id" db:"id"`
	Ver int    `json:"version" db:"version"` // Version conflicts with gorps optimistic locking.

	AppName  string `json:"-" db:"app_id"`
	ConfigID string `json:"-" db:"config_id"`
	SlugID   string `json:"-" db:"slug_id"`

	Description string    `json:"description" db:"description"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PreInsert implements a pre insert hook for the db interface
func (r *Release) PreInsert(s gorp.SqlExecutor) error {
	r.CreatedAt = Now()
	return nil
}

type ReleasesCreator interface {
	ReleasesCreate(*App, *Config, *Slug, string) (*Release, error)
}

type ReleasesFinder interface {
	ReleasesFindByApp(*App) ([]*Release, error)
	ReleasesFindByAppAndVersion(*App, int) (*Release, error)
	ReleasesLast(*App) (*Release, error)
}

// ReleaseesService represents a service for interacting with Releases.
type ReleasesService interface {
	ReleasesCreator
	ReleasesFinder
}

// releasesService is an implementation of the ReleasesRepository interface backed by
// a DB.
type releasesService struct {
	*db
	store *Store
	Manager
}

func (s *releasesService) ReleasesLast(app *App) (*Release, error) {
	return releasesLast(s.db, app.Name)
}

func (s *releasesService) ReleasesFindByApp(app *App) ([]*Release, error) {
	return releasesAllByAppName(s.db, app.Name)
}

func (s *releasesService) ReleasesFindByAppAndVersion(app *App, v int) (*Release, error) {
	return releasesFindByAppNameAndVersion(s.db, app.Name, v)
}

// Create creates the release, then sets the current process formation on the release.
func (s *releasesService) ReleasesCreate(app *App, config *Config, slug *Slug, desc string) (*Release, error) {
	r := &Release{
		AppName:     app.Name,
		ConfigID:    config.ID,
		SlugID:      slug.ID,
		Description: desc,
	}

	r, err := releasesCreate(s.db, r)
	if err != nil {
		return r, err
	}

	// Create a new formation for this release.
	formation, err := s.createFormation(r, slug)
	if err != nil {
		return nil, err
	}

	// Schedule the new release onto the cluster.
	if err := s.Manager.ScheduleRelease(r, config, slug, formation); err != nil {
		return r, err
	}

	return r, nil
}

func (s *releasesService) createFormation(release *Release, slug *Slug) (Formation, error) {
	// Get the old release, so we can copy the Formation.
	prev := release.Ver - 1
	last, err := releasesFindByAppNameAndVersion(s.db, release.AppName, prev)
	if err != nil {
		return nil, err
	}

	var existing Formation

	if last != nil {
		existing, err = s.store.ProcessesAll(last)
		if err != nil {
			return nil, err
		}
	}

	f := NewFormation(existing, slug.ProcessTypes)

	for _, p := range f {
		p.ReleaseID = release.ID

		if _, err := s.store.ProcessesCreate(p); err != nil {
			return f, err
		}
	}

	return f, nil
}

// ReleasesFindByAppNameAndVersion finds a specific version of a release for a
// given app.
func releasesFindByAppNameAndVersion(db *db, appName string, v int) (*Release, error) {
	var release Release

	if err := db.SelectOne(&release, `select * from releases where app_id = $1 and version = $2 limit 1`, appName, v); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &release, nil
}

// ReleasesCreate creates a new Release and inserts it into the database.
func releasesCreate(db *db, release *Release) (*Release, error) {
	t, err := db.Begin()
	if err != nil {
		return release, err
	}

	// Get the last release version for this app.
	v, err := releasesLastVersion(t, release.AppName)
	if err != nil {
		return release, err
	}

	// Increment the release version.
	release.Ver = v + 1

	if err := t.Insert(release); err != nil {
		return release, err
	}

	return release, t.Commit()
}

// ReleasesLastVersion returns the last ReleaseVersion for the given App. This
// function also ensures that the last release is locked until the transaction
// is commited, so the release version can be incremented atomically.
func releasesLastVersion(db interface {
	SelectOne(interface{}, string, ...interface{}) error
}, appName string) (version int, err error) {
	err = db.SelectOne(&version, `select version from releases where app_id = $1 order by version desc for update`, string(appName))

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return
}

// ReleasesLast returns the last Release for the given App.
func releasesLast(db *db, appName string) (*Release, error) {
	var release Release

	if err := db.SelectOne(&release, `select * from releases where app_id = $1 order by version desc limit 1`, appName); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &release, nil
}

// ReleasesFindByAppName finds the latest release for the given app.
func releasesAllByAppName(db *db, appName string) ([]*Release, error) {
	var rs []*Release
	return rs, db.Select(&rs, `select * from releases where app_id = $1 order by version desc`, appName)
}
