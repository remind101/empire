package cli_test

import "testing"

func TestCreate(t *testing.T) {
	run(t, []Command{
		{
			"apps",
			"",
		},
		{
			"create acme-inc",
			"Created acme-inc.",
		},
	})
}

func TestApps(t *testing.T) {
	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		{
			"apps",
			"acme-inc      Dec 31 17:01",
		},
	})
}
