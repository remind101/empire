package cli_test

import "testing"

func TestRollback(t *testing.T) {
	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
Status: Created new release v1 for acme-inc`,
		},
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
Status: Created new release v2 for acme-inc`,
		},
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
			`v1    Dec 31 17:01  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
v2    Dec 31 17:01  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
v3    Dec 31 17:01  Set FOO config vars
v4    Dec 31 17:01  Rollback to v1`,
		},
		{
			"env -a acme-inc",
			"",
		},
	})
}
