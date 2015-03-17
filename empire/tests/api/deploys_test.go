package api_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
	"github.com/remind101/empire/empire"
)

type DeployForm struct {
	Image struct {
		Repo string `json:"repo"`
		ID   string `json:"id"`
	} `json:"image"`
}

type Deploy struct {
	Release struct {
		ID string `json:"id"`
	} `json:"release"`
}

func TestDeploy(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustDeploy(t, c, DefaultImage)
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
