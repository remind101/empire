package empire

import (
	"context"
	"io"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
)

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	// A unique uuid that identifies this slug.
	ID string

	// The Docker image that this slug is for.
	Image image.Image

	// The raw Procfile that was extracted from the Docker image.
	Procfile []byte
}

// Formation returns a new Formation built from the extracted Procfile.
func (s *Slug) Formation() (Formation, error) {
	p, err := procfile.ParseProcfile(s.Procfile)
	if err != nil {
		return nil, err
	}

	return formationFromProcfile(p)
}

// slugsCreate inserts a Slug into the database.
func slugsCreate(db *gorm.DB, slug *Slug) (*Slug, error) {
	return slug, db.Create(slug).Error
}

// slugsService provides convenience methods for creating slugs.
type slugsService struct {
	*Empire
}

// SlugsCreateByImage creates a Slug for the given image.
func (s *slugsService) Create(ctx context.Context, db *gorm.DB, img image.Image, out io.Writer) (*Slug, error) {
	return slugsCreateByImage(ctx, db, s.ProcfileExtractor, img, out)
}

// SlugsCreateByImage first attempts to find a matching slug for the image. If
// it's not found, it will fallback to extracting the process types using the
// provided extractor, then create a slug.
func slugsCreateByImage(ctx context.Context, db *gorm.DB, e ProcfileExtractor, img image.Image, out io.Writer) (*Slug, error) {
	slug, err := slugsExtract(ctx, e, img, out)
	if err != nil {
		return slug, err
	}

	return slugsCreate(db, slug)
}

// SlugsExtract extracts the process types from the image, then returns a new
// Slug instance.
func slugsExtract(ctx context.Context, extractor ProcfileExtractor, img image.Image, out io.Writer) (*Slug, error) {
	slug := &Slug{
		Image: img,
	}

	p, err := extractor.Extract(ctx, img, out)
	if err != nil {
		return slug, err
	}

	slug.Procfile = p

	return slug, nil
}
