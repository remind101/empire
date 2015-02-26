package empire

import "database/sql/driver"

// SlugID represents the unique identifier of a Slug.
type SlugID string

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	ID           SlugID     `json:"id" db:"id"`
	Image        Image      `json:"image"`
	ProcessTypes CommandMap `json:"process_types" db:"process_types"`
}

// Scan implements the sql.Scanner interface.
func (id *SlugID) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*id = SlugID(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (id SlugID) Value() (driver.Value, error) {
	return driver.Value(string(id)), nil
}

// SlugsRepository represents an interface for creating and finding slugs.
type SlugsRepository interface {
	Create(*Slug) (*Slug, error)
	FindByID(SlugID) (*Slug, error)
	FindByImage(Image) (*Slug, error)
}

// NewSlugsRepository returns a new Repository instance.
func NewSlugsRepository(db DB) (SlugsRepository, error) {
	return &slugsRepository{db}, nil
}

// slugsRepository is a fake implementation of the Repository interface.
type slugsRepository struct {
	DB
}

// Create implements Repository Create.
func (r *slugsRepository) Create(slug *Slug) (*Slug, error) {
	return slug, r.DB.Insert(slug)
}

// FindByID implements Repository FindByID.
func (r *slugsRepository) FindByID(id SlugID) (*Slug, error) {
	var slug Slug

	if err := r.DB.SelectOne(&slug, `select * from slugs where id = $1`, string(id)); err != nil {
		return nil, err
	}

	return &slug, nil
}

func (r *slugsRepository) FindByImage(image Image) (*Slug, error) {
	var slug Slug

	if err := r.DB.SelectOne(&slug, `select * from slugs where image = $1`, image.String()); err != nil {
		return nil, err
	}

	return &slug, nil
}

// SlugsService is a service for interacting with slugs.
type SlugsService interface {
	// CreateByImage extracts process types from an image, then creates a
	// slug for it.
	CreateByImage(Image) (*Slug, error)
}

// slugsService is a base implementation of the SlugsService interface.
type slugsService struct {
	Repository SlugsRepository
	Extractor  Extractor
}

// NewSlugsService returns a new SlugsService instance.
func NewSlugsService(r SlugsRepository, e Extractor) (SlugsService, error) {
	return &slugsService{
		Repository: r,
		Extractor:  e,
	}, nil
}

// CreateByImageID extracts the process types from the image, then creates a new
// slug.
func (s *slugsService) CreateByImage(image Image) (*Slug, error) {
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
