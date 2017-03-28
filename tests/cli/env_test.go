package cli_test

import (
	"fmt"
	"io/ioutil"
	"os"
	"testing"
	"time"
)

func TestConfig(t *testing.T) {
	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		{
			"set RAILS_ENV=production -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			"RAILS_ENV=production",
		},
		{
			"set DATABASE_URL=postgres://localhost AUTH=foo -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"unset RAILS_ENV -a acme-inc",
			"Unset env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			`AUTH=foo
DATABASE_URL=postgres://localhost`,
		},
		{
			"set EMPTY_VAR= -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			`AUTH=foo
DATABASE_URL=postgres://localhost
EMPTY_VAR=`,
		},
	})
}

// TODO(ejholmes): This was disabled when switching to the ECS backend, since we
// no longer encode the release version in the process name.
func testUpdateConfigNewReleaseSameFormation(t *testing.T) {
	now(time.Now().AddDate(0, 0, -5))
	defer resetNow()

	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			"Deployed remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
		},
		{
			"dynos -a acme-inc",
			"web.1    running   5d  \"./bin/web\"",
		},
		{
			"scale web=2 -a acme-inc",
			"Scaled acme-inc to web=2:1X.",
		},
		{
			"dynos -a acme-inc",
			`web.1    running   5d  "./bin/web"
web.2    running   5d  "./bin/web"`,
		},
		{
			"set DATABASE_URL=postgres://localhost AUTH=foo -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"dynos -a acme-inc",
			`acme-inc.2.web.1    running   5d  "./bin/web"
acme-inc.2.web.2    running   5d  "./bin/web"`,
		},
	})
}

// Test to prevent a regression of https://github.com/remind101/empire/pull/612,
// where the releases weren't being sorted by version, so a random config was
// chosen.
func TestConfigConsistency(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"set FOO1=foo1 -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			"FOO1=foo1",
		},
		{
			"set FOO2=foo2 -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			"FOO1=foo1\nFOO2=foo2",
		},
		{
			"set FOO3=foo3 -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"env -a acme-inc",
			"FOO1=foo1\nFOO2=foo2\nFOO3=foo3",
		},
	})
}

func TestConfigNoop(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"set FOO=bar -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"releases -a acme-inc",
			`v1    Dec 31  2014  Deploy remind101/acme-inc:latest (fake)
v2    Dec 31  2014  Added (FOO) config var (fake)`,
		},
		{
			"set FOO=bar -a acme-inc",
			"Set env vars and restarted acme-inc.",
		},
		{
			"releases -a acme-inc",
			`v1    Dec 31  2014  Deploy remind101/acme-inc:latest (fake)
v2    Dec 31  2014  Added (FOO) config var (fake)
v3    Dec 31  2014  Made no changes to config vars (fake)`,
		},
	})
}

func TestEnvConfig(t *testing.T) {
	f, err := ioutil.TempFile(os.TempDir(), "acme-inc.env")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(f.Name())

	content := []byte("FOO=bar\\nmoarbar\nBAR=foo")
	f.Write(content)

	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			fmt.Sprintf("env-load %s -a acme-inc", f.Name()),
			fmt.Sprintf("Updated env vars from %s and restarted acme-inc.", f.Name()),
		},
		{
			"env -a acme-inc",
			"BAR=foo\nFOO=bar\\nmoarbar",
		},
	})
}
