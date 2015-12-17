package api_test

import (
	"testing"

	"github.com/remind101/empire/pkg/heroku"
)

func TestProcessesGet(t *testing.T) {
	c, s := NewTestClient(t)
	defer s.Close()

	mustDeploy(t, c, DefaultImage)

	q := 1
	mustFormationBatchUpdate(t, c, "acme-inc", []heroku.FormationBatchUpdateOpts{
		{
			Process:  "web",
			Quantity: &q,
		},
	})

	dynos, err := c.DynoList("acme-inc", nil)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := len(dynos), q; got != want {
		t.Errorf("DynoList => %d; want %d", got, want)
	}

	if got, want := dynos[0].Type, "web"; got != want {
		t.Errorf("dyno.Type => %s; want %s", got, want)
	}
}

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
