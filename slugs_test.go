package empire

import (
	"errors"
	"reflect"
	"testing"

	"github.com/remind101/empire/images"
	"github.com/remind101/empire/slugs"
)

func TestSlugsServiceCreateByImage(t *testing.T) {
	image := &images.Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	pm := slugs.ProcessMap{"web": "./web"}

	r := &mockSlugsRepository{}
	e := &mockExtractor{
		ExtractFunc: func(image *images.Image) (slugs.ProcessMap, error) {
			return pm, nil
		},
	}

	s := &slugsService{
		Repository: r,
		Extractor:  e,
	}

	slug, err := s.CreateByImage(image)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := slug.ProcessTypes, pm; !reflect.DeepEqual(got, want) {
		t.Fatal("Expected process types to be assigned to the slug")
	}

	if got, want := slug.Image, image; !reflect.DeepEqual(got, want) {
		t.Fatal("Expected the image to be assigned to the slug")
	}
}

func TestSlugsServiceCreateByImageFound(t *testing.T) {
	image := &images.Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	r := &mockSlugsRepository{
		FindByImageFunc: func(image *images.Image) (*slugs.Slug, error) {
			return &slugs.Slug{
				ID: "1234",
			}, nil
		},
	}
	e := &mockExtractor{
		ExtractFunc: func(image *images.Image) (slugs.ProcessMap, error) {
			t.Fatal("Expected Extract to not be called")
			return nil, nil
		},
	}

	s := &slugsService{
		Repository: r,
		Extractor:  e,
	}

	slug, err := s.CreateByImage(image)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := slug.ID, slugs.ID("1234"); got != want {
		t.Fatal("Expected a slug to be returned")
	}
}

func TestSlugsServiceCreateByImageFoundError(t *testing.T) {
	image := &images.Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	r := &mockSlugsRepository{
		FindByImageFunc: func(image *images.Image) (*slugs.Slug, error) {
			return &slugs.Slug{
				ID: "1234",
			}, errors.New("empire: shit")
		},
	}
	e := &mockExtractor{
		ExtractFunc: func(image *images.Image) (slugs.ProcessMap, error) {
			t.Fatal("Expected Extract to not be called")
			return nil, nil
		},
	}

	s := &slugsService{
		Repository: r,
		Extractor:  e,
	}

	if _, err := s.CreateByImage(image); err == nil {
		t.Fatal("Expected an error")
	}
}

type mockExtractor struct {
	ExtractFunc func(*images.Image) (slugs.ProcessMap, error)
}

func (e *mockExtractor) Extract(image *images.Image) (slugs.ProcessMap, error) {
	if e.ExtractFunc != nil {
		return e.ExtractFunc(image)
	}

	return slugs.ProcessMap{}, nil
}

type mockSlugsRepository struct {
	slugs.Repository // Just to satisfy the interface.

	CreateFunc      func(*slugs.Slug) (*slugs.Slug, error)
	FindByImageFunc func(*images.Image) (*slugs.Slug, error)
}

func (r *mockSlugsRepository) Create(slug *slugs.Slug) (*slugs.Slug, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(slug)
	}

	return slug, nil
}

func (r *mockSlugsRepository) FindByImage(image *images.Image) (*slugs.Slug, error) {
	if r.FindByImageFunc != nil {
		return r.FindByImageFunc(image)
	}

	return nil, nil
}

type mockSlugsService struct {
	SlugsService // Just to satisfy the interface.

	CreateByImageFunc func(*images.Image) (*slugs.Slug, error)
}

func (s *mockSlugsService) CreateByImage(image *images.Image) (*slugs.Slug, error) {
	if s.CreateByImageFunc != nil {
		return s.CreateByImageFunc(image)
	}

	return nil, nil
}
