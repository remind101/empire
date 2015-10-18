package cli_test

import "testing"

func TestRollback(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		DeployCommand("latest", "v2"),
		{
			"set FOO=bar -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"rollback v1 -a acme-inc",
			"Rolled back acme-inc to v1 as v4.",
		},
		{
			"releases -a acme-inc",
			`v1    Dec 31 17:01  Deploy remind101/acme-inc:latest
v2    Dec 31 17:01  Deploy remind101/acme-inc:latest
v3    Dec 31 17:01  Set FOO config var
v4    Dec 31 17:01  Rollback to v1`,
		},
		{
			"env -a acme-inc",
			"",
		},
	})
}
