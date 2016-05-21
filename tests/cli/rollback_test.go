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
			`v1    Dec 31  2014  Deploy remind101/acme-inc:latest (fake)
v2    Dec 31  2014  Deploy remind101/acme-inc:latest (fake)
v3    Dec 31  2014  Set FOO config var (fake)
v4    Dec 31  2014  Rollback to v1 (fake)`,
		},
		{
			"env -a acme-inc",
			"",
		},
	})
}
