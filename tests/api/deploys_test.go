package api_test

import (
	"io/ioutil"
	"testing"

	"github.com/remind101/empire/pkg/heroku"
)

type DeployForm struct {
	Image string
}

func TestDeploy(t *testing.T) {
	c := NewTestClient(t)
	defer c.Close()

	mustDeploy(t, c.Client, DefaultImage)
}

func mustDeploy(t testing.TB, c *heroku.Client, image string) {
	var (
		f DeployForm
	)

	f.Image = image

	if err := c.Post(ioutil.Discard, "/deploys", &f); err != nil {
		t.Fatal(err)
	}
}
