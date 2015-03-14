package empire

import (
	"database/sql"
	"database/sql/driver"
	"time"

	"gopkg.in/gorp.v1"
)

// ReleaseID represents the unique identifier for a Release.
type ReleaseID string

// Scan implements the sql.Scanner interface.
func (r *ReleaseID) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*r = ReleaseID(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (r ReleaseID) Value() (driver.Value, error) {
	return driver.Value(string(r)), nil
}

// ReleaseVersion represents the auto incremented human friendly version number of the
// release.
type ReleaseVersion int

// Release is a combination of a Config and a Slug, which form a deployable
// release.
type Release struct {
	ID  ReleaseID      `json:"id" db:"id"`
	Ver ReleaseVersion `json:"version" db:"version"` // Version conflicts with gorps optimistic locking.

	AppName  string `json:"-" db:"app_id"`
	ConfigID `json:"-" db:"config_id"`
	SlugID   `json:"-" db:"slug_id"`

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
	ReleasesFindByAppAndVersion(*App, ReleaseVersion) (*Release, error)
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
	DB
	ProcessesService
	Manager
}

func (s *releasesService) ReleasesLast(app *App) (*Release, error) {
	return ReleasesLast(s.DB, app.Name)
}

func (s *releasesService) ReleasesFindByApp(app *App) ([]*Release, error) {
	return ReleasesAllByAppName(s.DB, app.Name)
}

func (s *releasesService) ReleasesFindByAppAndVersion(app *App, v ReleaseVersion) (*Release, error) {
	return ReleasesFindByAppNameAndVersion(s.DB, app.Name, v)
}

// Create creates the release, then sets the current process formation on the release.
func (s *releasesService) ReleasesCreate(app *App, config *Config, slug *Slug, desc string) (*Release, error) {
	r := &Release{
		AppName:     app.Name,
		ConfigID:    config.ID,
		SlugID:      slug.ID,
		Description: desc,
	}

	r, err := ReleasesCreate(s.DB, r)
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
	prev := int(release.Ver) - 1
	last, err := ReleasesFindByAppNameAndVersion(s.DB, release.AppName, ReleaseVersion(prev))
	if err != nil {
		return nil, err
	}

	var existing Formation

	if last != nil {
		existing, err = s.ProcessesService.ProcessesAll(last)
		if err != nil {
			return nil, err
		}
	}

	f := NewFormation(existing, slug.ProcessTypes)

	for _, p := range f {
		p.ReleaseID = release.ID

		if _, err := s.ProcessesService.ProcessesCreate(p); err != nil {
			return f, err
		}
	}

	return f, nil
}

// ReleasesFindByAppNameAndVersion finds a specific version of a release for a
// given app.
func ReleasesFindByAppNameAndVersion(db Queryier, appName string, v ReleaseVersion) (*Release, error) {
	var release Release

	if err := db.SelectOne(&release, `select * from releases where app_id = $1 and version = $2 limit 1`, appName, int(v)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return &release, nil
}

// ReleasesCreate creates a new Release and inserts it into the database.
func ReleasesCreate(db DB, release *Release) (*Release, error) {
	t, err := db.Begin()
	if err != nil {
		return release, err
	}

	// Get the last release version for this app.
	v, err := ReleasesLastVersion(t, release.AppName)
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
func ReleasesLastVersion(db Queryier, appName string) (version ReleaseVersion, err error) {
	err = db.SelectOne(&version, `select version from releases where app_id = $1 order by version desc for update`, appName)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return
}

// ReleasesLast returns the last Release for the given App.
func ReleasesLast(db Queryier, appName string) (*Release, error) {
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
func ReleasesAllByAppName(db Queryier, appName string) ([]*Release, error) {
	var rs []*Release
	return rs, db.Select(&rs, `select * from releases where app_id = $1 order by version desc`, appName)
}
