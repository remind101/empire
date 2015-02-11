package units

import (
	"sort"
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

func TestCreateRelease(t *testing.T) {
	ps := NewService(newRepository())

	err := ps.CreateRelease(buildRelease("api", "1", slugs.ProcessMap{
		"web":      "./web",
		"worker":   "./worker",
		"consumer": "./consumer",
	}))
	if err != nil {
		t.Fatal(err)
	}

	testUnitsEql(t, ps.Repository, "api", []string{
		"api.web release=1 count=1",
		"api.worker release=1 count=0",
		"api.consumer release=1 count=0",
	})

	err = ps.CreateRelease(buildRelease("api", "2", slugs.ProcessMap{
		"web":    "./web",
		"worker": "./worker",
	}))
	if err != nil {
		t.Fatal(err)
	}

	testUnitsEql(t, ps.Repository, "api", []string{
		"api.web release=2 count=1",
		"api.worker release=2 count=0",
	})
}

func TestScale(t *testing.T) {
	ps := NewService(newRepository())

	rel := buildRelease("api", "1", slugs.ProcessMap{
		"web": "./web",
	})

	err := ps.CreateRelease(rel)
	if err != nil {
		t.Fatal(err)
	}

	_, err = ps.Scale("api", "web", 3)
	if err != nil {
		t.Fatal(err)
	}

	testUnitsEql(t, ps.Repository, "api", []string{
		"api.web release=1 count=3",
	})
}

func TestDelete(t *testing.T) {
	ps := NewService(newRepository())

	rel := buildRelease("api", "1", slugs.ProcessMap{
		"web":      "./web",
		"worker":   "./worker",
		"consumer": "./consumer",
	})

	err := ps.CreateRelease(rel)
	if err != nil {
		t.Fatal(err)
	}

	// Delete just one process type
	err = ps.Delete("api", "consumer")
	if err != nil {
		t.Fatal(err)
	}

	testUnitsEql(t, ps.Repository, "api", []string{
		"api.web release=1 count=1",
		"api.worker release=1 count=0",
	})

	// Delete all processes for a repo
	err = ps.Delete("api", "")
	if err != nil {
		t.Fatal(err)
	}

	testUnitsEql(t, ps.Repository, "api", []string{})
}

func buildRelease(appID string, releaseID string, proctypes slugs.ProcessMap) *releases.Release {
	return &releases.Release{
		ID:      releases.ID(releaseID),
		Version: "v1",
		App: &apps.App{
			ID: apps.ID(appID),
		},
		Config: &configs.Config{
			Vars: configs.Vars{
				"RAILS_ENV": "test",
			},
		},
		Slug: &slugs.Slug{
			Image: &slugs.Image{
				ID: "abcd",
			},
			ProcessTypes: proctypes,
		},
	}
}

func testUnitsEql(t *testing.T, s Repository, id apps.ID, unitStrings []string) {
	foundUnits, err := s.FindByApp(id)
	if err != nil {
		t.Fatal(err)
	}

	foundStrings := make([]string, len(foundUnits))
	for i, u := range foundUnits {
		foundStrings[i] = u.String()
	}

	sort.Strings(unitStrings)
	sort.Strings(foundStrings)

	if got, want := len(foundStrings), len(unitStrings); got != want {
		t.Errorf("len(s.FindByApp(\"%s\")) => %v; want %v", id, got, want)
	}

	for i, def := range unitStrings {
		if got, want := foundStrings[i], def; got != want {
			t.Errorf("s.FindByApp(\"%s\")[%v] => %v; want %v", id, i, got, want)
		}
	}
}
