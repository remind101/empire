package cli_test

import (
	"regexp"
	"testing"
)

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
			"acme-inc    Dec 31  2014",
		},
	})
}

func TestAppInfo(t *testing.T) {
	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		{
			"info -a acme-inc",
			regexp.MustCompile("Name: acme-inc\nID:   [0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\n"),
		},
	})
}
