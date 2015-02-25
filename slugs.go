package empire

import (
	"database/sql"

	"github.com/lib/pq/hstore"
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
func NewSlugsRepository(db DB) (SlugsRepository, error) {
	return &slugsRepository{db}, nil
}

type dbSlug struct {
	ID           string        `db:"id"`
	ImageRepo    string        `db:"image_repo"`
	ImageID      string        `db:"image_id"`
	ProcessTypes hstore.Hstore `db:"process_types"`
}

// slugsRepository is a fake implementation of the Repository interface.
type slugsRepository struct {
	DB
}

// Create implements Repository Create.
func (r *slugsRepository) Create(slug *Slug) (*Slug, error) {
	s := fromSlug(slug)

	if err := r.DB.Insert(s); err != nil {
		return slug, err
	}

	return toSlug(s, slug), nil
}

// FindByID implements Repository FindByID.
func (r *slugsRepository) FindByID(id SlugID) (*Slug, error) {
	var s dbSlug

	if err := r.DB.SelectOne(&s, `select * from slugs where id = $1`, string(id)); err != nil {
		return nil, err
	}

	return toSlug(&s, nil), nil
}

func (r *slugsRepository) FindByImage(image *Image) (*Slug, error) {
	var s dbSlug

	if err := r.DB.SelectOne(&s, `select * from slugs where image_repo = $1 and image_id = $2`, string(image.Repo), string(image.ID)); err != nil {
		return nil, err
	}

	return toSlug(&s, nil), nil
}

func fromSlug(slug *Slug) *dbSlug {
	pt := make(map[string]sql.NullString)

	for k, v := range slug.ProcessTypes {
		pt[string(k)] = sql.NullString{
			Valid:  true,
			String: string(v),
		}
	}

	return &dbSlug{
		ID:        string(slug.ID),
		ImageRepo: string(slug.Image.Repo),
		ImageID:   string(slug.Image.ID),
		ProcessTypes: hstore.Hstore{
			Map: pt,
		},
	}
}

func toSlug(s *dbSlug, slug *Slug) *Slug {
	if slug == nil {
		slug = &Slug{}
	}

	cm := make(CommandMap)

	for k, v := range s.ProcessTypes.Map {
		cm[ProcessType(k)] = Command(v.String)
	}

	slug.ID = SlugID(s.ID)
	slug.Image = &Image{
		Repo: Repo(s.ImageRepo),
		ID:   s.ImageRepo,
	}
	slug.ProcessTypes = cm

	return slug
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
func NewSlugsService(r SlugsRepository, e Extractor) (SlugsService, error) {
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
