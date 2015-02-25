package empire

import (
	"code.google.com/p/go-uuid/uuid"

	"github.com/remind101/empire/stores"
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

type slugsRepository struct {
	s stores.Store
}

func NewSlugsRepository() (SlugsRepository, error) {
	return &slugsRepository{stores.NewMemStore()}, nil
}

func NewEtcdSlugsRepository(ns string) (SlugsRepository, error) {
	s, err := stores.NewEtcdStore(ns)
	if err != nil {
		return nil, err
	}
	return &slugsRepository{s}, nil
}

// Create implements Repository Create.
func (r *slugsRepository) Create(slug *Slug) (*Slug, error) {
	slug.ID = SlugID(uuid.NewRandom())

	if err := r.s.Set(string(slug.ID), slug); err != nil {
		return nil, err
	}

	return slug, nil
}

// FindByID implements Repository FindByID.
func (r *slugsRepository) FindByID(id SlugID) (*Slug, error) {
	s := &Slug{}

	if ok, err := r.s.Get(string(id), s); err != nil || !ok {
		return nil, err
	}

	return s, nil
}

func (r *slugsRepository) FindByImage(image *Image) (*Slug, error) {
	slugs := make([]*Slug, 0)

	if err := r.s.List("", &slugs); err != nil {
		return nil, err
	}

	for _, slug := range slugs {
		if *slug.Image == *image {
			return slug, nil
		}
	}

	return nil, nil
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
