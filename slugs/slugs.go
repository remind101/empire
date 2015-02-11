package slugs

import (
	"strconv"

	"github.com/remind101/empire/repos"
)

// ProcessType represents the type of a given process/command.
type ProcessType string

// Command represents the actual shell command that gets executed for a given
// ProcessType.
type Command string

// ProcessMap represents a map of ProcessType -> Command.
type ProcessMap map[ProcessType]Command

// Image represents a container image, which is tied to a repository.
type Image struct {
	Repo repos.Repo
	ID   string
}

// Slug represents a container image with the extracted ProcessTypes.
type Slug struct {
	ID           string
	Image        *Image
	ProcessTypes ProcessMap
}

// Extractor represents an object that can extract the process types from an
// image.
type Extractor interface {
	// Extract takes a repo in the form `remind101/r101-api`, and an image
	// id, and extracts the process types from the image.
	Extract(*Image) (ProcessMap, error)
}

// extractor is a fake implementation of the Extractor interface.
type extractor struct{}

// Extract implements Extractor Extract.
func (e *extractor) Extract(image *Image) (ProcessMap, error) {
	pm := make(ProcessMap)

	// Just return some fake processes.
	pm[ProcessType("web")] = Command("./bin/web")

	return pm, nil
}

// Repository represents an interface for creating and finding slugs.
type Repository interface {
	Create(*Slug) (*Slug, error)
	FindByID(id string) (*Slug, error)
	FindByImage(*Image) (*Slug, error)
}

// slugsRepository is a fake implementation of the Repository interface.
type slugsRepository struct {
	// map[slug.ID]*Slug
	slugs map[string]*Slug
	id    int
}

// newRepository returns a new slugsRepository instance.
func newRepository() *slugsRepository {
	return &slugsRepository{
		slugs: make(map[string]*Slug),
	}
}

// Create implements Repository Create.
func (r *slugsRepository) Create(slug *Slug) (*Slug, error) {
	r.id++
	slug.ID = strconv.Itoa(r.id)
	r.slugs[slug.ID] = slug
	return slug, nil
}

// FindByID implements Repository FindByID.
func (r *slugsRepository) FindByID(id string) (*Slug, error) {
	return r.slugs[id], nil
}

func (r *slugsRepository) FindByImage(image *Image) (*Slug, error) {
	for _, slug := range r.slugs {
		if *slug.Image == *image {
			return slug, nil
		}
	}

	return nil, nil
}

func (r *slugsRepository) Reset() {
	r.slugs = make(map[string]*Slug)
	r.id = 0
}

// Service is a service for extracting process types then creating a new
// Slug.
type Service struct {
	Repository
	Extractor Extractor
}

// CreateByImageID extracts the process types from the image, then creates a new
// slug.
func (s *Service) CreateByImage(image *Image) (*Slug, error) {
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
