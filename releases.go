package empire

import (
	"fmt"
	"time"

	"code.google.com/p/go-uuid/uuid"

	"github.com/remind101/empire/stores"
)

// ReleaseID represents the unique identifier for a Release.
type ReleaseID string

// ReleaseVersion represents the auto incremented human friendly version number of the
// release.
type ReleaseVersion string

// Release is a combination of a Config and a Slug, which form a deployable
// release.
type Release struct {
	ID        ReleaseID      `json:"id"`
	Version   ReleaseVersion `json:"version"`
	App       *App           `json:"app"`
	Config    *Config        `json:"config"`
	Formation *Formation     `json:"formation"`
	Slug      *Slug          `json:"slug"`
	CreatedAt time.Time      `json:"created_at"`
}

// ReleaseRepository is an interface that can be implemented for storing and
// retrieving releases.
type ReleasesRepository interface {
	Create(*Release) (*Release, error)
	FindByAppName(AppName) ([]*Release, error)
	Head(AppName) (*Release, error)
}

type releasesRepository struct {
	s            stores.Store
	genTimestamp func() time.Time
}

// NewReleasesRepository returns a new repository backed by an in memory store.
func NewReleasesRepository() *releasesRepository {
	return &releasesRepository{s: stores.NewMemStore()}
}

// NewEtcdReleasesRepository returns a new repository backed by etcd
func NewEtcdReleasesRepository(ns string) (*releasesRepository, error) {
	s, err := stores.NewEtcdStore(ns)
	if err != nil {
		return nil, err
	}

	return &releasesRepository{s: s}, nil
}

// Generates a repository that stubs out the CreatedAt field.
func newFakeRepository() *releasesRepository {
	r := NewReleasesRepository()
	r.genTimestamp = func() time.Time {
		return time.Date(2014, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return r
}

func (r *releasesRepository) Create(release *Release) (*Release, error) {
	app := release.App

	createdAt := time.Now()
	if r.genTimestamp != nil {
		createdAt = r.genTimestamp()
	}

	// If a version is found, version is set to it, otherwise it stays untouched
	var version int
	if _, err := r.s.Get(r.keyVersion(app.Name), &version); err != nil {
		return nil, err
	}
	version++

	release.ID = ReleaseID(uuid.NewRandom())
	release.Version = ReleaseVersion(fmt.Sprintf("v%d", version))
	release.CreatedAt = createdAt.UTC()

	// Set the current version
	if err := r.s.Set(r.keyVersion(app.Name), &version); err != nil {
		return nil, err
	}

	// Set the head release
	if err := r.s.Set(r.keyHead(app.Name), &release); err != nil {
		return nil, err
	}

	// Set the release
	if err := r.s.Set(r.keyByRelease(app.Name, release.Version), &release); err != nil {
		return nil, err
	}

	return release, nil
}

func (r *releasesRepository) FindByAppName(appName AppName) ([]*Release, error) {
	rels := make([]*Release, 0)

	if err := r.s.List(r.keyByApp(appName), &rels); err != nil {
		return nil, err
	}

	return rels, nil
}

func (r *releasesRepository) Head(appName AppName) (*Release, error) {
	rel := &Release{}

	if ok, err := r.s.Get(r.keyHead(appName), rel); err != nil || !ok {
		return nil, err
	}

	return rel, nil
}

func (r *releasesRepository) keyVersion(appName AppName) string {
	return fmt.Sprintf("%s/version", appName)
}

func (r *releasesRepository) keyHead(appName AppName) string {
	return fmt.Sprintf("%s/head", appName)
}

func (r *releasesRepository) keyByApp(appName AppName) string {
	return fmt.Sprintf("%s/versions/", appName)
}

func (r *releasesRepository) keyByRelease(appName AppName, v ReleaseVersion) string {
	return fmt.Sprintf("%s/versions/%s", appName, v)
}

// ReleaseesService represents a service for interacting with Releases.
type ReleasesService interface {
	// Create creates a new release.
	Create(*App, *Config, *Slug) (*Release, error)
}

// releasesService is a base implementation of the ReleasesService interface.
type releasesService struct {
	Repository           ReleasesRepository
	FormationsRepository FormationsRepository
	Manager              Manager
}

// NewReleasesService returns a new ReleasesService instance.
func NewReleasesService(options Options, m Manager) (ReleasesService, error) {
	return &releasesService{
		Repository:           NewReleasesRepository(),
		FormationsRepository: NewFormationsRepository(),
		Manager:              m,
	}, nil
}

// Create creates the release, then sets the current process formation on the release.
func (s *releasesService) Create(app *App, config *Config, slug *Slug) (*Release, error) {
	// Create a new formation for this release.
	formation, err := s.createFormation(app, slug)
	if err != nil {
		return nil, err
	}

	r := &Release{
		App:       app,
		Config:    config,
		Slug:      slug,
		Formation: formation,
	}

	r, err = s.Repository.Create(r)
	if err != nil {
		return r, err
	}

	// Schedule the new release onto the cluster.
	if err := s.Manager.ScheduleRelease(r); err != nil {
		return r, err
	}

	return s.Repository.Create(r)
}

func (s *releasesService) createFormation(app *App, slug *Slug) (*Formation, error) {
	// Get the old release, so we can copy the Formation.
	old, err := s.Repository.Head(app.Name)
	if err != nil {
		return nil, err
	}

	var p ProcessMap
	if old != nil {
		p = old.Formation.Processes
	}

	formation := &Formation{
		Processes: NewProcessMap(p, slug.ProcessTypes),
	}

	return s.FormationsRepository.Create(formation)
}
