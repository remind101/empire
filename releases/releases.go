package releases

import (
	"fmt"
	"strconv"
	"time"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
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
	ID        ID
	Version   Version
	App       *apps.App
	Config    *configs.Config
	Slug      *slugs.Slug
	CreatedAt time.Time
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
	releases     map[apps.ID][]*Release
	versions     map[apps.ID]int
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

func (p *repository) Create(app *apps.App, config *configs.Config, slug *slugs.Slug) (*Release, error) {
	p.id++

	createdAt := time.Now()
	if p.genTimestamp != nil {
		createdAt = p.genTimestamp()
	}

	version := 1
	if v, ok := p.versions[app.ID]; ok {
		version = v
	}

	r := &Release{
		ID:        ID(strconv.Itoa(p.id)),
		Version:   Version(fmt.Sprintf("v%d", version)),
		App:       app,
		Config:    config,
		Slug:      slug,
		CreatedAt: createdAt.UTC(),
	}

	p.versions[app.ID] = version + 1
	p.releases[app.ID] = append(p.releases[app.ID], r)

	return r, nil
}

func (p *repository) FindByAppID(id apps.ID) ([]*Release, error) {
	if set, ok := p.releases[id]; ok {
		return set, nil
	}

	return []*Release{}, nil
}

func (p *repository) Head(id apps.ID) (*Release, error) {
	set, ok := p.releases[id]
	if !ok {
		return nil, nil
	}

	return set[len(set)-1], nil
}

// Service provides methods for interacting with releases.
type Service struct {
	Repository
}

// NewService returns a new Service instance.
func NewService(r Repository) *Service {
	if r == nil {
		r = newRepository()
	}

	return &Service{Repository: r}
}
