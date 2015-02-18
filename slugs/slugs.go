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

// ProcessMap represents a map of processes.Type -> processes.Command.
type ProcessMap map[processes.Type]processes.Command

// ID represents the unique identifier of a Slug.
type ID string

// Slug represents a container image with the extracted processes.Type.
type Slug struct {
	ID           ID            `json:"id"`
	Image        *images.Image `json:"image"`
	ProcessTypes ProcessMap    `json:"process_types"`
}

// Repository represents an interface for creating and finding slugs.
type Repository interface {
	Create(*Slug) (*Slug, error)
	FindByID(ID) (*Slug, error)
	FindByImage(*images.Image) (*Slug, error)
}

// NewRepository returns a new Repository instance.
func NewRepository() (Repository, error) {
	return nil, nil
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

// Service is a service for extracting process types then creating a new
// Slug.
type Service struct {
	Repository
	Extractor Extractor
}

// NewService returns a new Service instance.
func NewService(r Repository, e Extractor) *Service {
	if r == nil {
		r = newRepository()
	}

	if e == nil {
		e = newExtractor()
	}

	return &Service{
		Repository: r,
		Extractor:  e,
	}
}

// CreateByImageID extracts the process types from the image, then creates a new
// slug.
func (s *Service) CreateByImage(image *images.Image) (*Slug, error) {
	if slug, err := s.Repository.FindByImage(image); slug != nil {
		return slug, err
	}

	slug := &Slug{
		Image: image,
	}

	pt, err := s.Extractor.Extract(image)
	if err != nil {
		return slug, err
	}

	slug.ProcessTypes = pt

	return s.Repository.Create(slug)
}
