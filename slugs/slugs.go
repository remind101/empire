package slugs

import (
	"strconv"
	"sync"

	"github.com/remind101/empire/images"
	"github.com/remind101/empire/processes"
)

var (
	// DefaultProcfilePath is the default path where Procfiles will be
	// extracted from the container.
	DefaultProcfilePath = "/home/app/Procfile"
)

// ID represents the unique identifier of a Slug.
type ID string

// Slug represents a container image with the extracted processes.Type.
type Slug struct {
	ID           ID                   `json:"id"`
	Image        *images.Image        `json:"image"`
	ProcessTypes processes.CommandMap `json:"process_types"`
}

// Repository represents an interface for creating and finding slugs.
type Repository interface {
	Create(*Slug) (*Slug, error)
	FindByID(ID) (*Slug, error)
	FindByImage(*images.Image) (*Slug, error)
}

// NewRepository returns a new Repository instance.
func NewRepository() (Repository, error) {
	return newRepository(), nil
}

// repository is a fake implementation of the Repository interface.
type repository struct {
	id int

	sync.RWMutex
	slugs map[ID]*Slug
}

// newRepository returns a new repository instance.
func newRepository() *repository {
	return &repository{
		slugs: make(map[ID]*Slug),
	}
}

// Create implements Repository Create.
func (r *repository) Create(slug *Slug) (*Slug, error) {
	r.Lock()
	defer r.Unlock()

	r.id++
	slug.ID = ID(strconv.Itoa(r.id))
	r.slugs[slug.ID] = slug
	return slug, nil
}

// FindByID implements Repository FindByID.
func (r *repository) FindByID(id ID) (*Slug, error) {
	r.RLock()
	defer r.RUnlock()

	return r.slugs[id], nil
}

func (r *repository) FindByImage(image *images.Image) (*Slug, error) {
	r.RLock()
	defer r.RUnlock()

	for _, slug := range r.slugs {
		if *slug.Image == *image {
			return slug, nil
		}
	}

	return nil, nil
}

func (r *repository) Reset() {
	r.Lock()
	defer r.Unlock()

	r.slugs = make(map[ID]*Slug)
	r.id = 0
}
