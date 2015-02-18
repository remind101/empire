package formations

import (
	"testing"

	"github.com/remind101/empire/apps"
)

func TestFindFormation(t *testing.T) {
	formations := make(Formations)

	f := findFormation(formations, "web")
	if got, want := f.Count, 1; got != want {
		t.Fatalf("Count => %v; want %v", got, want)
	}

	if got, want := len(formations), 1; got != want {
		t.Fatal("Expected the new formation to be added")
	}

	f = findFormation(formations, "web")
	if got, want := len(formations), 1; got != want {
		t.Fatal("Expected the old formation to be fetched")
	}
}

func TestServiceScale(t *testing.T) {
	r := newRepository()
	s := &Service{Repository: r}

	app := &apps.App{ID: "1234"}
	if f, err := s.Scale(app, "web", 2); err == nil {
		if got, want := f.Count, 2; got != want {
			t.Fatalf("Count => %v; want %v", got, want)
		}
	} else {
		t.Fatal(err)
	}
}
