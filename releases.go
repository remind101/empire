package empire

import (
	"database/sql"
	"time"
)

// ReleaseID represents the unique identifier for a Release.
type ReleaseID string

// ReleaseVersion represents the auto incremented human friendly version number of the
// release.
type ReleaseVersion int

// Release is a combination of a Config and a Slug, which form a deployable
// release.
type Release struct {
	ID        ReleaseID      `json:"id"`
	Version   ReleaseVersion `json:"version"`
	CreatedAt time.Time      `json:"created_at"`

	App       *App      `json:"app"`
	Config    *Config   `json:"config"`
	Formation Formation `json:"formation"`
	Slug      *Slug     `json:"slug"`
}

// ReleaseRepository is an interface that can be implemented for storing and
// retrieving releases.
type ReleasesRepository interface {
	Create(*Release) (*Release, error)
	FindByAppName(AppName) ([]*Release, error)
	Head(AppName) (*Release, error)
}

// NewReleasesRepository is a factory method that returns a new Repository.
func NewReleasesRepository(db DB) (ReleasesRepository, error) {
	return &releasesRepository{db}, nil
}

// dbRelease is a db representation of a release.
type dbRelease struct {
	ID      *string `db:"id"`
	AppID   string  `db:"app_id"`
	Version int64   `db:"ver"`
}

// releasesRepository is an implementation of the ReleasesRepository interface backed by
// a DB.
type releasesRepository struct {
	DB
}

func (r *releasesRepository) Create(release *Release) (*Release, error) {
	rl := fromRelease(release)

	t, err := r.DB.Begin()
	if err != nil {
		return release, err
	}

	var version int64
	if err := t.SelectOne(&version, `select ver from releases where app_id = $1 order by ver desc`, string(release.App.Name)); err != nil {
		if err == sql.ErrNoRows {
			version = 1
		} else {
			return release, err
		}
	}

	rl.Version = version

	if err := t.Insert(rl); err != nil {
		return release, err
	}

	if err := t.Commit(); err != nil {
		return release, err
	}

	return toRelease(rl, release), nil
}

func (r *releasesRepository) Head(appName AppName) (*Release, error) {
	return headRelease(r.DB, appName)
}

func (r *releasesRepository) FindByAppName(appName AppName) ([]*Release, error) {
	var rs []*dbRelease

	if err := r.DB.Select(rs, `select * from releases where app_id = $1 order by id desc limit 1`, string(appName)); err != nil {
		return nil, nil
	}

	var releases []*Release

	for _, r := range rs {
		releases = append(releases, toRelease(r, nil))
	}

	return releases, nil
}

func headRelease(db Queryier, appName AppName) (*Release, error) {
	var rl dbRelease

	if err := db.SelectOne(&rl, `select * from releases where app_id = $1 order by id desc limit 1`, string(appName)); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return toRelease(&rl, nil), nil
}

func fromRelease(release *Release) *dbRelease {
	id := string(release.ID)

	return &dbRelease{
		ID:      &id,
		AppID:   string(release.App.Name),
		Version: int64(release.Version),
	}
}

func toRelease(r *dbRelease, release *Release) *Release {
	if release == nil {
		release = &Release{}
	}

	release.ID = ReleaseID(*r.ID)
	release.App = &App{Name: AppName(r.AppID)}
	release.Version = ReleaseVersion(r.Version)

	return release
}

// ReleaseesService represents a service for interacting with Releases.
type ReleasesService interface {
	// Create creates a new release.
	Create(*App, *Config, *Slug) (*Release, error)

	// Find existing releases for an app
	FindByApp(*App) ([]*Release, error)

	// Find current release for an app
	Head(*App) (*Release, error)
}

// releasesService is a base implementation of the ReleasesService interface.
type releasesService struct {
	ReleasesRepository
	ProcessesRepository
	Manager
}

// NewReleasesService returns a new ReleasesService instance.
func NewReleasesService(r ReleasesRepository, p ProcessesRepository, m Manager) (ReleasesService, error) {
	return &releasesService{
		ReleasesRepository:  r,
		ProcessesRepository: p,
		Manager:             m,
	}, nil
}

// Create creates the release, then sets the current process formation on the release.
func (s *releasesService) Create(app *App, config *Config, slug *Slug) (*Release, error) {
	r := &Release{
		App:    app,
		Config: config,
		Slug:   slug,
	}

	r, err := s.ReleasesRepository.Create(r)
	if err != nil {
		return r, err
	}

	// Create a new formation for this release.
	formation, err := s.createFormation(r)
	if err != nil {
		return nil, err
	}

	r.Formation = formation

	// Schedule the new release onto the cluster.
	if err := s.Manager.ScheduleRelease(r); err != nil {
		return r, err
	}

	return r, nil
}

func (s *releasesService) FindByApp(a *App) ([]*Release, error) {
	return s.ReleasesRepository.FindByAppName(a.Name)
}

func (s *releasesService) Head(app *App) (*Release, error) {
	return s.ReleasesRepository.Head(app.Name)
}

func (s *releasesService) createFormation(release *Release) (Formation, error) {
	// Get the old release, so we can copy the Formation.
	last, err := s.ReleasesRepository.Head(release.App.Name)
	if err != nil {
		return nil, err
	}

	var existing Formation
	if last != nil {
		existing = last.Formation
	}

	f := NewFormation(existing, release.Slug.ProcessTypes)

	for t, p := range f {
		p.Release = release

		if _, _, err := s.ProcessesRepository.Create(t, p); err != nil {
			return f, err
		}
	}

	return f, nil
}
