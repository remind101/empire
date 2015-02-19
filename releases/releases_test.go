package releases

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/slugs"
)

func TestServiceCreate(t *testing.T) {
	f := formations.NewService(nil)
	s := NewService(nil, f)

	app := &apps.App{ID: "1234"}
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
