package empire_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire"
)

type DeployForm struct {
	Image struct {
		Repo string `json:"repo"`
		ID   string `json:"id"`
	} `json:"image"`
}

type Deploy struct {
	Release struct {
		ID      string `json:"id"`
		Version int    `json:"version"`
	} `json:"release"`
}

func TestDeploy(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	d := mustDeploy(t, c, DefaultImage)

	if got, want := d.Release.Version, 1; got != want {
		t.Fatal("Version => %v; want %v", got, want)
	}
}

func mustDeploy(t testing.TB, c *heroku.Client, image empire.Image) Deploy {
	var (
		f DeployForm
		d Deploy
	)

	f.Image.Repo = string(image.Repo)
	f.Image.ID = string(image.ID)

	if err := c.Post(&d, "/deploys", &f); err != nil {
		t.Fatal(err)
	}

	return d
}
