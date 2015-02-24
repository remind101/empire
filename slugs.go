package empire

import (
	"strconv"
	"sync"
)

// SlugID represents the unique identifier of a Slug.
type SlugID string

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	ID           SlugID     `json:"id"`
	Image        *Image     `json:"image"`
	ProcessTypes CommandMap `json:"process_types"`
}

// SlugsRepository represents an interface for creating and finding slugs.
type SlugsRepository interface {
	Create(*Slug) (*Slug, error)
	FindByID(SlugID) (*Slug, error)
	FindByImage(*Image) (*Slug, error)
}

// NewSlugsRepository returns a new Repository instance.
func NewSlugsRepository() (SlugsRepository, error) {
	return newSlugsRepository(), nil
}

// slugsRepository is a fake implementation of the Repository interface.
type slugsRepository struct {
	id int

	sync.RWMutex
	slugs map[SlugID]*Slug
}

// newSlugsRepository returns a new repository instance.
func newSlugsRepository() *slugsRepository {
	return &slugsRepository{
		slugs: make(map[SlugID]*Slug),
	}
}

// Create implements Repository Create.
func (r *slugsRepository) Create(slug *Slug) (*Slug, error) {
	r.Lock()
	defer r.Unlock()

	r.id++
	slug.ID = SlugID(strconv.Itoa(r.id))
	r.slugs[slug.ID] = slug
	return slug, nil
}

// FindByID implements Repository FindByID.
func (r *slugsRepository) FindByID(id SlugID) (*Slug, error) {
	r.RLock()
	defer r.RUnlock()

	return r.slugs[id], nil
}

func (r *slugsRepository) FindByImage(image *Image) (*Slug, error) {
	r.RLock()
	defer r.RUnlock()

	for _, slug := range r.slugs {
		if *slug.Image == *image {
			return slug, nil
		}
	}

	return nil, nil
}

func (r *slugsRepository) Reset() {
	r.Lock()
	defer r.Unlock()

	r.slugs = make(map[SlugID]*Slug)
	r.id = 0
}

// SlugsService is a service for interacting with slugs.
type SlugsService interface {
	// CreateByImage extracts process types from an image, then creates a
	// slug for it.
	CreateByImage(*Image) (*Slug, error)
}

// slugsService is a base implementation of the SlugsService interface.
type slugsService struct {
	Repository SlugsRepository
	Extractor  Extractor
}

// NewSlugsService returns a new SlugsService instance.
func NewSlugsService(options Options) (SlugsService, error) {
	r, err := NewSlugsRepository()
	if err != nil {
		return nil, err
	}

	e, err := NewExtractor(
		options.Docker.Socket,
		options.Docker.Registry,
		options.Docker.CertPath,
	)
	if err != nil {
		return nil, err
	}

	return &slugsService{
		Repository: r,
		Extractor:  e,
	}, nil
}

// CreateByImageID extracts the process types from the image, then creates a new
// slug.
func (s *slugsService) CreateByImage(image *Image) (*Slug, error) {
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
