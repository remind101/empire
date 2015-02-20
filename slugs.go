package empire

import (
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/slugs"
)

// SlugsService is a service for interacting with slugs.
type SlugsService interface {
	slugs.Repository

	// CreateByImage extracts process types from an image, then creates a
	// slug for it.
	CreateByImage(*images.Image) (*slugs.Slug, error)
}

// slugsService is a base implementation of the SlugsService interface.
type slugsService struct {
	slugs.Repository
	Extractor slugs.Extractor
}

// NewSlugsService returns a new SlugsService instance.
func NewSlugsService(r slugs.Repository, e slugs.Extractor) SlugsService {
	return &slugsService{
		Repository: r,
		Extractor:  e,
	}
}

// CreateByImageID extracts the process types from the image, then creates a new
// slug.
func (s *slugsService) CreateByImage(image *images.Image) (*slugs.Slug, error) {
	if slug, err := s.Repository.FindByImage(image); slug != nil {
		return slug, err
	}

	slug := &slugs.Slug{
		Image: image,
	}

	pt, err := s.Extractor.Extract(image)
	if err != nil {
		return slug, err
	}

	slug.ProcessTypes = pt

	return s.Repository.Create(slug)
}
