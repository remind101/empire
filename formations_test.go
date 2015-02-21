package empire

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/processes"
	"github.com/remind101/empire/slugs"
)

func TestFormationsServiceGetOrCreate(t *testing.T) {
	var set bool

	app := &apps.App{}
	slug := &slugs.Slug{
		ProcessTypes: slugs.ProcessMap{
			"web":    "./bin/web",
			"worker": "sidekiq",
		},
	}

	r := &mockFormationsRepository{
		GetFunc: func(app *apps.App) (formations.Formations, error) {
			return nil, ErrNoFormation
		},
		SetFunc: func(app *apps.App, f formations.Formations) error {
			set = true

			if _, ok := f["web"]; !ok {
				t.Fatal("Expected a web formation")
			}

			if _, ok := f["worker"]; !ok {
				t.Fatal("Expected a worker formation")
			}

			return nil
		},
	}
	s := &formationsService{
		Repository: r,
	}

	if _, err := s.GetOrCreate(app, slug); err != nil {
		t.Fatal(err)
	}

	if got, want := set, true; got != want {
		t.Fatal("Expected a new formation to be set")
	}
}

func TestFormationsServiceScaleNoFormation(t *testing.T) {
	app := &apps.App{}

	r := &mockFormationsRepository{
		SetFunc: func(app *apps.App, f formations.Formations) error {
			return nil
		},
	}
	s := &formationsService{
		Repository: r,
	}

	if _, err := s.Scale(app, "web", 5); err != nil {
		if got, want := err, ErrNoFormation; got != want {
			t.Fatalf("error => %s; want %s", got, want)
		}
	} else {
		t.Fatal("Expected an error")
	}
}

func TestFormationsServiceScaleNoProcessType(t *testing.T) {
	app := &apps.App{}

	r := &mockFormationsRepository{
		GetFunc: func(app *apps.App) (formations.Formations, error) {
			return formations.Formations{}, nil
		},
	}
	s := &formationsService{
		Repository: r,
	}

	if _, err := s.Scale(app, "web", 5); err != nil {
		if got, want := err, ErrInvalidProcessType; got != want {
			t.Fatalf("error => %s; want %s", got, want)
		}
	} else {
		t.Fatal("Expected an error")
	}
}

type mockFormationsRepository struct {
	SetFunc func(*apps.App, formations.Formations) error
	GetFunc func(*apps.App) (formations.Formations, error)
}

func (r *mockFormationsRepository) Set(app *apps.App, f formations.Formations) error {
	if r.SetFunc != nil {
		return r.SetFunc(app, f)
	}

	return nil
}

func (r *mockFormationsRepository) Get(app *apps.App) (formations.Formations, error) {
	if r.GetFunc != nil {
		return r.GetFunc(app)
	}

	return nil, nil
}

type mockFormationsService struct {
	mockFormationsRepository

	GetOrCreateFunc func(*apps.App, *slugs.Slug) (formations.Formations, error)
	ScaleFunc       func(*apps.App, processes.Type, int) (*formations.Formation, error)
}

func (s *mockFormationsService) GetOrCreate(app *apps.App, slug *slugs.Slug) (formations.Formations, error) {
	if s.GetOrCreateFunc != nil {
		return s.GetOrCreateFunc(app, slug)
	}

	return nil, nil
}

func (s *mockFormationsService) Scale(app *apps.App, pt processes.Type, count int) (*formations.Formation, error) {
	if s.ScaleFunc != nil {
		return s.ScaleFunc(app, pt, count)
	}

	return nil, nil
}
