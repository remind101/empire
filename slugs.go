package empire

import (
	"fmt"

	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/procfile"
	"golang.org/x/net/context"
)

// Slug represents a container image with the extracted ProcessType.
type Slug struct {
	// The Docker image that this slug is for.
	Image image.Image

	// The raw Procfile that was extracted from the Docker image.
	Procfile []byte
}

// ParsedProcfile returns the parsed Procfile.
func (s *Slug) ParsedProcfile() (procfile.Procfile, error) {
	return procfile.ParseProcfile(s.Procfile)
}

// Formation returns a new Formation built from the extracted Procfile.
func (s *Slug) Formation() (Formation, error) {
	p, err := s.ParsedProcfile()
	if err != nil {
		return nil, err
	}

	return formationFromProcfile(p)
}

// slugsService provides convenience methods for creating slugs.
type slugsService struct {
	*Empire
}

// SlugsCreateByImage creates a Slug for the given image.
func (s *slugsService) Create(ctx context.Context, img image.Image, w *DeploymentStream) (*Slug, error) {
	return slugsCreateByImage(ctx, s.ImageRegistry, img, w)
}

// SlugsCreateByImage first attempts to find a matching slug for the image. If
// it's not found, it will fallback to extracting the process types using the
// provided extractor, then create a slug.
func slugsCreateByImage(ctx context.Context, r ImageRegistry, img image.Image, w *DeploymentStream) (*Slug, error) {
	var (
		slug Slug
		err  error
	)

	slug.Image, err = r.Resolve(ctx, img, w.Stream)
	if err != nil {
		return nil, fmt.Errorf("resolving %s: %v", img, err)
	}

	slug.Procfile, err = r.ExtractProcfile(ctx, slug.Image, w.Stream)
	if err != nil {
		return nil, fmt.Errorf("extracting Procfile from %s: %v", slug.Image, err)
	}

	return &slug, nil
}
