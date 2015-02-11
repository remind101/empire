package slugs

import (
	"reflect"
	"testing"
)

func TestRepository(t *testing.T) {
	r := newRepository()

	if got, want := len(r.slugs), 0; got != want {
		t.Fatal("Expected no slugs")
	}

	image := &Image{Repo: "remind101/r101-api", ID: "1234"}
	if slug, err := r.Create(&Slug{Image: image}); err == nil {
		expected := &Slug{
			ID:    "1",
			Image: image,
		}
		if got, want := slug, expected; !reflect.DeepEqual(got, want) {
			t.Fatalf("Create => %q; want %q")
		}
	} else {
		t.Fatal(err)
	}

	if got, want := len(r.slugs), 1; got != want {
		t.Fatal("Slugs count %d; want %d", got, want)
	}

	if slug, err := r.FindByImage(&Image{Repo: "remind101/r101-api", ID: "1234"}); err == nil {
		if slug == nil {
			t.Fatal("Expected a slug to be returned")
		}
	} else {
		t.Fatal(err)
	}
}

func TestService_CreateByImageID(t *testing.T) {
	s := &Service{
		Repository: newRepository(),
		Extractor:  &extractor{},
	}

	image := &Image{
		Repo: "ejholmes/docker-statsd",
		ID:   "1234",
	}

	slug, err := s.CreateByImage(image)
	if err != nil {
		t.Fatal(err)
	}

	expected := &Slug{ID: "1", Image: image, ProcessTypes: ProcessMap{"web": "./bin/web"}}
	if got, want := slug, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Slug => %q; want %q", got, want)
	}
}

func TestService_CreateByImageID_AlreadyExists(t *testing.T) {
	s := &Service{
		Repository: newRepository(),
		Extractor:  &extractor{},
	}

	image := &Image{
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

	expected := &Slug{ID: "1", Image: image, ProcessTypes: ProcessMap{"web": "./bin/web"}}
	if got, want := slug, expected; !reflect.DeepEqual(got, want) {
		t.Fatalf("Slug => %q; want %q", got, want)
	}
}
