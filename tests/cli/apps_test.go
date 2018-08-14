package cli_test

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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
			regexp.MustCompile("Name: acme-inc\nID: [0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}\n"),
		},
	})
}

func TestCreateDestroyCreate(t *testing.T) {
	cli := newCLI(t)
	defer cli.Close()

	cli.Auth(t)
	cli.RunCommands(t, []Command{
		{
			"apps",
			"",
		},
		{
			"create acme-inc",
			"Created acme-inc.",
		},
	})

	cmd := cli.Command("destroy", "acme-inc")
	cmd.Stdin = strings.NewReader("acme-inc\n")
	assert.NoError(t, cmd.Run())

	cli.RunCommands(t, []Command{
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
