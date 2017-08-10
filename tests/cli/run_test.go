package cli_test

import (
	"testing"

	"github.com/remind101/empire/empiretest"
)

func TestRunDetached(t *testing.T) {
	empiretest.SkipCI(t)

	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"run -d migration -a acme-inc",
			"Ran `migration` on acme-inc as run, detached.",
		},
	})
}

func TestRunAttached(t *testing.T) {
	empiretest.SkipCI(t)

	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"run migration -a acme-inc",
			"Attaching to container\nFake output for `[migration]` on acme-inc",
		},
	})
}
