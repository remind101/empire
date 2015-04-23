package empire

import (
	"database/sql"
	"fmt"
	"io"
)

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	ID           string     `json:"id" db:"id"`
	Image        Image      `json:"image"`
	ProcessTypes CommandMap `json:"process_types" db:"process_types"`
}

func (s *store) SlugsCreate(slug *Slug) (*Slug, error) {
	return slugsCreate(s.db, slug)
}

func (s *store) SlugsFind(id string) (*Slug, error) {
	return slugsFind(s.db, id)
}

func (s *store) SlugsFindByImage(image Image) (*Slug, error) {
	return slugsFindByImage(s.db, image)
}

// SlugsCreate inserts a Slug into the database.
func slugsCreate(db *db, slug *Slug) (*Slug, error) {
	return slug, db.Insert(slug)
}

// SlugsFind finds a slug by id.
func slugsFind(db *db, id string) (*Slug, error) {
	return slugsFindBy(db, "id", id)
}

// SlugsFindByImage finds a slug by image.
func slugsFindByImage(db *db, image Image) (*Slug, error) {
	return slugsFindBy(db, "image", image.String())
}

// SlugsFindBy finds a slug by a field.
func slugsFindBy(db *db, field string, value interface{}) (*Slug, error) {
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

// slugsService provides convenience methods for creating slugs.
type slugsService struct {
	store     *store
	extractor Extractor
	resolver  Resolver
}

// SlugsCreateByImage creates a Slug for the given image.
func (s *slugsService) SlugsCreateByImage(image Image, output io.Writer) (*Slug, error) {
	return slugsCreateByImage(s.store, s.extractor, s.resolver, image, output)
}

// SlugsCreateByImage first attempts to find a matching slug for the image. If
// it's not found, it will fallback to extracting the process types using the
// provided extractor, then create a slug.
func slugsCreateByImage(store *store, e Extractor, r Resolver, image Image, output io.Writer) (*Slug, error) {
	image, err := r.Resolve(image, output)
	if err != nil {
		return nil, err
	}

	slug, err := store.SlugsFindByImage(image)
	if err != nil {
		return slug, err
	}

	if slug != nil {
		return slug, nil
	}

	slug, err = slugsExtract(e, image)
	if err != nil {
		return slug, err
	}

	return store.SlugsCreate(slug)
}

// SlugsExtract extracts the process types from the image, then returns a new
// Slug instance.
func slugsExtract(e Extractor, image Image) (*Slug, error) {
	slug := &Slug{
		Image: image,
	}

	pt, err := e.Extract(image)
	if err != nil {
		return slug, err
	}

	slug.ProcessTypes = pt

	return slug, nil
}
