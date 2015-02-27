package empire

import (
	"errors"
	"reflect"
	"testing"
)

func TestSlugsServiceCreateByImage(t *testing.T) {
	image := Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	pm := CommandMap{"web": "./web"}

	r := &mockSlugsRepository{}
	e := &mockExtractor{
		ExtractFunc: func(image Image) (CommandMap, error) {
			return pm, nil
		},
	}

	s := &slugsService{
		SlugsRepository: r,
		Extractor:       e,
	}

	slug, err := s.CreateByImage(image)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := slug.ProcessTypes, pm; !reflect.DeepEqual(got, want) {
		t.Fatal("Expected process types to be assigned to the slug")
	}

	if got, want := slug.Image, image; got != want {
		t.Fatal("Expected the image to be assigned to the slug")
	}
}

func TestSlugsServiceCreateByImageFound(t *testing.T) {
	image := Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	r := &mockSlugsRepository{
		FindByImageFunc: func(image Image) (*Slug, error) {
			return &Slug{
				ID: "1234",
			}, nil
		},
	}
	e := &mockExtractor{
		ExtractFunc: func(image Image) (CommandMap, error) {
			t.Fatal("Expected Extract to not be called")
			return nil, nil
		},
	}

	s := &slugsService{
		SlugsRepository: r,
		Extractor:       e,
	}

	slug, err := s.CreateByImage(image)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := slug.ID, SlugID("1234"); got != want {
		t.Fatal("Expected a slug to be returned")
	}
}

func TestSlugsServiceCreateByImageFoundError(t *testing.T) {
	image := Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	r := &mockSlugsRepository{
		FindByImageFunc: func(image Image) (*Slug, error) {
			return &Slug{
				ID: "1234",
			}, errors.New("empire: shit")
		},
	}
	e := &mockExtractor{
		ExtractFunc: func(image Image) (CommandMap, error) {
			t.Fatal("Expected Extract to not be called")
			return nil, nil
		},
	}

	s := &slugsService{
		SlugsRepository: r,
		Extractor:       e,
	}

	if _, err := s.CreateByImage(image); err == nil {
		t.Fatal("Expected an error")
	}
}

type mockExtractor struct {
	ExtractFunc func(Image) (CommandMap, error)
}

func (e *mockExtractor) Extract(image Image) (CommandMap, error) {
	if e.ExtractFunc != nil {
		return e.ExtractFunc(image)
	}

	return CommandMap{}, nil
}

type mockSlugsRepository struct {
	SlugsRepository // Just to satisfy the interface.

	CreateFunc      func(*Slug) (*Slug, error)
	FindByImageFunc func(Image) (*Slug, error)
}

func (r *mockSlugsRepository) Create(slug *Slug) (*Slug, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(slug)
	}

	return slug, nil
}

func (r *mockSlugsRepository) FindByImage(image Image) (*Slug, error) {
	if r.FindByImageFunc != nil {
		return r.FindByImageFunc(image)
	}

	return nil, nil
}

type mockSlugsService struct {
	SlugsService // Just to satisfy the interface.

	CreateByImageFunc func(Image) (*Slug, error)
}

func (s *mockSlugsService) CreateByImage(image Image) (*Slug, error) {
	if s.CreateByImageFunc != nil {
		return s.CreateByImageFunc(image)
	}

	return nil, nil
}
