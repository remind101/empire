package api_test

import (
	"testing"

	"github.com/remind101/empire/pkg/heroku"
)

func TestFormationBatchUpdate(t *testing.T) {
	c := newClient(t)
	defer c.Close()

	mustDeploy(t, c.Client, DefaultImage)

	q := 2
	f := mustFormationBatchUpdate(t, c.Client, "acme-inc", []heroku.FormationBatchUpdateOpts{
		{
			Process:  "web",
			Quantity: &q,
		},
	})

	if got, want := f[0].Quantity, 2; got != want {
		t.Fatalf("Quantity => %d; want %d", got, want)
	}
}

func mustFormationBatchUpdate(t testing.TB, c *heroku.Client, appName string, updates []heroku.FormationBatchUpdateOpts) []heroku.Formation {
	f, err := c.FormationBatchUpdate(appName, updates, "")
	if err != nil {
		t.Fatal(err)
	}

	return f
}
