package empire

import (
	"io"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/image"
	"golang.org/x/net/context"
)

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	ID           string
	Image        image.Image
	ProcessTypes CommandMap
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
	return slugsCreateByImage(ctx, db, s.ExtractProcfile, img, out)
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
func slugsExtract(ctx context.Context, extract ProcfileExtractor, img image.Image, out io.Writer) (*Slug, error) {
	slug := &Slug{
		Image: img,
	}

	p, err := extract(ctx, img, out)
	if err != nil {
		return slug, err
	}

	slug.ProcessTypes = commandMapFromProcfile(p)

	return slug, nil
}
