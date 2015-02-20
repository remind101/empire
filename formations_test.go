package empire

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/formations"
)

func TestFindFormation(t *testing.T) {
	fmtns := make(formations.Formations)

	f := findFormation(fmtns, "web")
	if got, want := f.Count, 1; got != want {
		t.Fatalf("Count => %v; want %v", got, want)
	}

	if got, want := len(fmtns), 1; got != want {
		t.Fatal("Expected the new formation to be added")
	}

	f = findFormation(fmtns, "web")
	if got, want := len(fmtns), 1; got != want {
		t.Fatal("Expected the old formation to be fetched")
	}
}

func TestFormationsServiceScale(t *testing.T) {
	s, err := NewFormationsService(DefaultOptions)
	if err != nil {
		t.Fatal(err)
	}

	app := &apps.App{Name: "abcd"}
	if f, err := s.Scale(app, "web", 2); err == nil {
		if got, want := f.Count, 2; got != want {
			t.Fatalf("Count => %v; want %v", got, want)
		}
	} else {
		t.Fatal(err)
	}
}
