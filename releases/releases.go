package releases

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/slugs"
)

// ID represents the unique identifier for a Release.
type ID string

// Version represents the auto incremented human friendly version number of the
// release.
type Version string

// Release is a combination of a Config and a Slug, which form a deployable
// release.
type Release struct {
	ID        ID                             `json:"id"`
	Version   Version                        `json:"version"`
	App       *apps.App                      `json:"app"`
	Config    *configs.Config                `json:"config"`
	Formation []*formations.CommandFormation `json:"formation"`
	Slug      *slugs.Slug                    `json:"slug"`
	CreatedAt time.Time                      `json:"created_at"`
}

// ReleaseRepository is an interface that can be implemented for storing and
// retrieving releases.
type Repository interface {
	Create(*apps.App, *configs.Config, *slugs.Slug) (*Release, error)
	FindByAppID(apps.ID) ([]*Release, error)
	Head(apps.ID) (*Release, error)
}

// repository is an in-memory implementation of a Repository
type repository struct {
	sync.RWMutex
	releases map[apps.ID][]*Release
	versions map[apps.ID]int

	genTimestamp func() time.Time
	id           int
}

// Create a new repository
func newRepository() *repository {
	return &repository{
		releases: make(map[apps.ID][]*Release),
		versions: make(map[apps.ID]int),
	}
}

// Generates a repository that stubs out the CreatedAt field.
func newFakeRepository() *repository {
	r := newRepository()
	r.genTimestamp = func() time.Time {
		return time.Date(2014, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
	return r
}

func (r *repository) Create(app *apps.App, config *configs.Config, slug *slugs.Slug) (*Release, error) {
	r.Lock()
	defer r.Unlock()

	r.id++

	createdAt := time.Now()
	if r.genTimestamp != nil {
		createdAt = r.genTimestamp()
	}

	version := 1
	if v, ok := r.versions[app.ID]; ok {
		version = v
	}

	release := &Release{
		ID:        ID(strconv.Itoa(r.id)),
		Version:   Version(fmt.Sprintf("v%d", version)),
		App:       app,
		Config:    config,
		Slug:      slug,
		CreatedAt: createdAt.UTC(),
	}

	r.versions[app.ID] = version + 1
	r.releases[app.ID] = append(r.releases[app.ID], release)

	return release, nil
}

func (r *repository) FindByAppID(id apps.ID) ([]*Release, error) {
	r.RLock()
	defer r.RUnlock()

	if set, ok := r.releases[id]; ok {
		return set, nil
	}

	return []*Release{}, nil
}

func (r *repository) Head(id apps.ID) (*Release, error) {
	r.RLock()
	defer r.RUnlock()

	set, ok := r.releases[id]
	if !ok {
		return nil, nil
	}

	return set[len(set)-1], nil
}

// Service provides methods for interacting with releases.
type Service struct {
	Repository
	FormationsService *formations.Service
}

// NewService returns a new Service instance.
func NewService(r Repository, f *formations.Service) *Service {
	if r == nil {
		r = newRepository()
	}

	return &Service{
		Repository:        r,
		FormationsService: f,
	}
}

func (s *Service) Create(app *apps.App, config *configs.Config, slug *slugs.Slug) (*Release, error) {
	r, err := s.Repository.Create(app, config, slug)
	if err != nil {
		return r, err
	}

	// Get the currently configured process formation.
	fmtns, err := s.FormationsService.Get(app)
	if err != nil {
		return r, err
	}

	for _, f := range fmtns {
		cmd, found := slug.ProcessTypes[f.ProcessType]
		if !found {
			// TODO Update the formation?
			continue
		}

		r.Formation = append(r.Formation, &formations.CommandFormation{
			Formation: f,
			Command:   cmd,
		})
	}

	return r, nil
}
