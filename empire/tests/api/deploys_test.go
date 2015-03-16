package api_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
)

type DeployForm struct {
	Image struct {
		Repo string `json:"repo"`
		Tag  string `json:"tag"`
	} `json:"image"`
}

type Deploy struct {
	Release struct {
		ID          string `json:"id"`
		Version     int    `json:"version"`
		Description string `json:"description"`
	} `json:"release"`
}

func TestDeploy(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	d := mustDeploy(t, c, DefaultImage)

	if got, want := d.Release.Version, 1; got != want {
		t.Fatalf("Version => %v; want %v", got, want)
	}

	if got, want := d.Release.Description, "Deploy "+DefaultImage.String(); got != want {
		t.Fatalf("Description => %v; want %v", got, want)
	}
}

func mustDeploy(t testing.TB, c *heroku.Client, image empire.Image) Deploy {
	var (
		f DeployForm
		d Deploy
	)

	f.Image.Repo = string(image.Repo)
	f.Image.Tag = string(image.Tag)

	if err := c.Post(&d, "/deploys", &f); err != nil {
		t.Fatal(err)
	}

	return d
}
