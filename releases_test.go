package empire

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/slugs"
)

func TestReleasesServiceCreate(t *testing.T) {
	f := formations.NewService(nil)
	s := NewReleasesService(nil, f)

	app := &apps.App{Name: "api"}
	config := &configs.Config{}
	slug := &slugs.Slug{
		ProcessTypes: slugs.ProcessMap{
			"web": "./bin/web",
		},
	}

	r, err := s.Create(app, config, slug)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Formation) != 1 {
		t.Fatal("Expected an initial process formation")
	}
}
