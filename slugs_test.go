package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/images"
	"github.com/remind101/empire/slugs"
)

func TestSlugsServiceCreateByImage(t *testing.T) {
	s, err := NewSlugsService(DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}

	image := &images.Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	slug, err := s.CreateByImage(image)
	if err != nil {
		t.Fatal(err)
	}

	expected := &slugs.Slug{ID: "1", Image: image, ProcessTypes: slugs.ProcessMap{"web": "./bin/web"}}
	if got, want := slug, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Slug => %q; want %q", got, want)
	}
}

func TestSlugsServiceCreateByImageAlreadyExists(t *testing.T) {
	s, err := NewSlugsService(DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}

	image := &images.Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	if _, err := s.CreateByImage(image); err != nil {
		t.Fatal(err)
	}

	slug, err := s.CreateByImage(image)
	if err != nil {
		t.Fatal(err)
	}

	expected := &slugs.Slug{ID: "1", Image: image, ProcessTypes: slugs.ProcessMap{"web": "./bin/web"}}
	if got, want := slug, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Slug => %q; want %q", got, want)
	}
}
