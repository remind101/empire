package empire

import (
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/slugs"
)

// SlugsService is a service for interacting with slugs.
type SlugsService interface {
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
func NewSlugsService(options Options) (SlugsService, error) {
	r, err := slugs.NewRepository()
	if err != nil {
		return nil, err
	}

	e, err := slugs.NewExtractor(
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
