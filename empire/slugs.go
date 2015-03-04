package empire

import (
	"database/sql"
	"database/sql/driver"
	"fmt"

	"github.com/fsouza/go-dockerclient"
)

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

type SlugsCreator interface {
	SlugsCreate(*Slug) (*Slug, error)
	SlugsCreateByImage(Image, *docker.AuthConfigurations) (*Slug, error)
}

type SlugsFinder interface {
	SlugsFind(SlugID) (*Slug, error)
	SlugsFindByImage(Image) (*Slug, error)
}

type SlugsService interface {
	SlugsCreator
	SlugsFinder
}

// slugsService is a fake implementation of the Repository interface.
type slugsService struct {
	DB
	extractor Extractor
}

func (s *slugsService) SlugsCreate(slug *Slug) (*Slug, error) {
	return SlugsCreate(s.DB, slug)
}

func (s *slugsService) SlugsFind(id SlugID) (*Slug, error) {
	return SlugsFind(s.DB, id)
}

func (s *slugsService) SlugsFindByImage(image Image) (*Slug, error) {
	return SlugsFindByImage(s.DB, image)
}

func (s *slugsService) SlugsCreateByImage(image Image, auth *docker.AuthConfigurations) (*Slug, error) {
	return SlugsCreateByImage(s.DB, s.extractor, image, auth)
}

// SlugsCreateByImage first attempts to find a matching slug for the image. If
// it's not found, it will fallback to extracting the process types using the
// provided extractor, then create a slug.
func SlugsCreateByImage(db DB, e Extractor, image Image, auth *docker.AuthConfigurations) (*Slug, error) {
	slug, err := SlugsFindByImage(db, image)
	if err != nil {
		return slug, err
	}

	if slug != nil {
		return slug, nil
	}

	slug, err = SlugsExtract(e, image, auth)
	if err != nil {
		return slug, err
	}

	return SlugsCreate(db, slug)
}

// SlugsExtract extracts the process types from the image, then returns a new
// Slug instance.
func SlugsExtract(e Extractor, image Image, auth *docker.AuthConfigurations) (*Slug, error) {
	slug := &Slug{
		Image: image,
	}

	pt, err := e.Extract(image, auth)
	if err != nil {
		return slug, err
	}

	slug.ProcessTypes = pt

	return slug, nil
}

// SlugsCreate inserts a Slug into the database.
func SlugsCreate(db Inserter, slug *Slug) (*Slug, error) {
	return slug, db.Insert(slug)
}

// SlugsFind finds a slug by id.
func SlugsFind(db Queryier, id SlugID) (*Slug, error) {
	return SlugsFindBy(db, "id", string(id))
}

// SlugsFindByImage finds a slug by image.
func SlugsFindByImage(db Queryier, image Image) (*Slug, error) {
	return SlugsFindBy(db, "image", image.String())
}

// SlugsFindBy finds a slug by a field.
func SlugsFindBy(db Queryier, field string, value interface{}) (*Slug, error) {
	var slug Slug

	q := fmt.Sprintf(`select * from slugs where %s = $1`, field)
	if err := db.SelectOne(&slug, q, value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &slug, nil
}
