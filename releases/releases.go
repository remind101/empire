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
	ID        ID                    `json:"id"`
	Version   Version               `json:"version"`
	App       *apps.App             `json:"app"`
	Config    *configs.Config       `json:"config"`
	Formation *formations.Formation `json:"formation"`
	Slug      *slugs.Slug           `json:"slug"`
	CreatedAt time.Time             `json:"created_at"`
}

// ReleaseRepository is an interface that can be implemented for storing and
// retrieving releases.
type Repository interface {
	Create(*Release) (*Release, error)
	FindByAppID(apps.Name) ([]*Release, error)
	Head(apps.Name) (*Release, error)
}

// NewRepository is a factory method that returns a new Repository.
func NewRepository() Repository {
	return newRepository()
}

// repository is an in-memory implementation of a Repository
type repository struct {
	sync.RWMutex
	releases map[apps.Name][]*Release
	versions map[apps.Name]int

	genTimestamp func() time.Time
	id           int
}

// Create a new repository
func newRepository() *repository {
	return &repository{
		releases: make(map[apps.Name][]*Release),
		versions: make(map[apps.Name]int),
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

func (r *repository) Create(release *Release) (*Release, error) {
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

	release.ID = ID(strconv.Itoa(r.id))
	release.Version = Version(fmt.Sprintf("v%d", version))
	release.CreatedAt = createdAt.UTC()

	r.versions[app.Name] = version + 1
	r.releases[app.Name] = append(r.releases[app.Name], release)

	return release, nil
}

func (r *repository) FindByAppID(id apps.Name) ([]*Release, error) {
	r.RLock()
	defer r.RUnlock()

	if set, ok := r.releases[id]; ok {
		return set, nil
	}

	return []*Release{}, nil
}

func (r *repository) Head(id apps.Name) (*Release, error) {
	r.RLock()
	defer r.RUnlock()

	set, ok := r.releases[id]
	if !ok {
		return nil, nil
	}

	return set[len(set)-1], nil
}
