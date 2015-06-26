package cli_test

import "testing"

func TestRunDetached(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"run -d migration -a acme-inc",
			"Ran `migration` on acme-inc as run, detached.",
		},
	})
}

func TestRunAttached(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"run migration -a acme-inc",
			"Fake output for `migration` on acme-inc",
		},
	})
}
