package api_test

import (
	"io/ioutil"
	"testing"

	"github.com/bgentry/heroku-go"
)

type DeployForm struct {
	Image string
}

func TestDeploy(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustDeploy(t, c, DefaultImage)
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
