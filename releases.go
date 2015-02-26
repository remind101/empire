package empire

import (
	"fmt"
	"strconv"
	"sync"
	"time"
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
	Formation Formation      `json:"formation"`
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

// NewReleasesRepository is a factory method that returns a new Repository.
func NewReleasesRepository() ReleasesRepository {
	return newReleasesRepository()
}

// releasesRepository is an in-memory implementation of a Repository
type releasesRepository struct {
	sync.RWMutex
	releases map[AppName][]*Release
	versions map[AppName]int

	genTimestamp func() time.Time
	id           int
}

// Create a new repository
func newReleasesRepository() *releasesRepository {
	return &releasesRepository{
		releases: make(map[AppName][]*Release),
		versions: make(map[AppName]int),
	}
}

// Generates a repository that stubs out the CreatedAt field.
func newFakeRepository() *releasesRepository {
	r := newReleasesRepository()
	r.genTimestamp = func() time.Time {
		return time.Date(2014, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return r
}

func (r *releasesRepository) Create(release *Release) (*Release, error) {
	r.Lock()
	defer r.Unlock()

	r.id++

	app := release.App

	createdAt := time.Now()
	if r.genTimestamp != nil {
		createdAt = r.genTimestamp()
	}

	version := 1
	if v, ok := r.versions[app.Name]; ok {
		version = v
	}

	release.ID = ReleaseID(strconv.Itoa(r.id))
	release.Version = ReleaseVersion(fmt.Sprintf("v%d", version))
	release.CreatedAt = createdAt.UTC()

	r.versions[app.Name] = version + 1
	r.releases[app.Name] = append(r.releases[app.Name], release)

	return release, nil
}

func (r *releasesRepository) FindByAppName(id AppName) ([]*Release, error) {
	r.RLock()
	defer r.RUnlock()

	if set, ok := r.releases[id]; ok {
		return set, nil
	}

	return []*Release{}, nil
}

func (r *releasesRepository) Head(id AppName) (*Release, error) {
	r.RLock()
	defer r.RUnlock()

	set, ok := r.releases[id]
	if !ok {
		return nil, nil
	}

	return set[len(set)-1], nil
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
func NewReleasesService(options Options, p ProcessesRepository, m Manager) (ReleasesService, error) {
	return &releasesService{
		ReleasesRepository:  NewReleasesRepository(),
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

	return s.ReleasesRepository.Create(r)
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
