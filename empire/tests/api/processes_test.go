package api_test

import (
	"testing"

	"github.com/bgentry/heroku-go"
)

func TestProcessesPost(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustDeploy(t, c, DefaultImage)
	a := false

	if _, err := c.DynoCreate("acme-inc", "bash", &heroku.DynoCreateOpts{
		Attach: &a,
		Env: &map[string]string{
			"COLUMNS": "178",
			"LINES":   "43",
			"TERM":    "xterm-256color",
		},
	}); err != nil {
		t.Fatal(err)
	}
}
