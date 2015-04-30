package api_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
)

type DeployForm struct {
	Image string
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

func mustDeploy(t testing.TB, c *heroku.Client, image string) Deploy {
	var (
		f DeployForm
		d Deploy
	)

	f.Image = image

	if err := c.Post(&d, "/deploys", &f); err != nil {
		t.Fatal(err)
	}

	return d
}
