package empire

import "github.com/jinzhu/gorm"

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	ID           string
	Image        Image
	ProcessTypes CommandMap
}

// SlugsCreate persists the slug.
func (s *store) SlugsCreate(slug *Slug) (*Slug, error) {
	return slugsCreate(s.db, slug)
}

// SlugsCreate inserts a Slug into the database.
func slugsCreate(db *gorm.DB, slug *Slug) (*Slug, error) {
	return slug, db.Create(slug).Error
}

// slugsService provides convenience methods for creating slugs.
type slugsService struct {
	store     *store
	extractor Extractor
	resolver  Resolver
}

// SlugsCreateByImage creates a Slug for the given image.
func (s *slugsService) SlugsCreateByImage(image Image, out chan Event) (*Slug, error) {
	return slugsCreateByImage(s.store, s.extractor, s.resolver, image, out)
}

// SlugsCreateByImage first attempts to find a matching slug for the image. If
// it's not found, it will fallback to extracting the process types using the
// provided extractor, then create a slug.
func slugsCreateByImage(store *store, e Extractor, r Resolver, image Image, out chan Event) (*Slug, error) {
	_, err := r.Resolve(image, out)
	if err != nil {
		return nil, err
	}

	slug, err := slugsExtract(e, image)
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
