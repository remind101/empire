package cli_test

import "testing"

func TestDeploy(t *testing.T) {
	run(t, []Command{
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
Status: Created new release v1 for acme-inc
Status: Finished processing events for release v1 of acme-inc`,
		},
		{
			"releases -a acme-inc",
			"v1    Dec 31  2014  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2 (fake)",
		},
		{
			"deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2 -m important",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
Status: Created new release v2 for acme-inc
Status: Finished processing events for release v2 of acme-inc`,
		},
		{
			"releases -a acme-inc",
			"v1    Dec 31  2014  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2 (fake)\nv2    Dec 31  2014  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2 (fake: 'important')",
		},
		{
			"create my-app",
			"Created my-app.",
		},
		{
			"deploy -a my-app remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc
345c7524bc96: Pulling image (9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2
Status: Created new release v1 for my-app
Status: Finished processing events for release v1 of my-app`,
		},
		{
			"releases -a my-app",
			"v1    Dec 31  2014  Deploy remind101/acme-inc:9ea71ea5abe676f117b2c969a6ea3c1be8ed4098d2118b1fd9ea5a5e59aa24f2 (fake)",
		},
	})
}

func TestDeploy_NoTag(t *testing.T) {
	run(t, []Command{
		{
			"deploy remind101/acme-inc",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (latest) from remind101/acme-inc
345c7524bc96: Pulling image (latest) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:latest
Status: Created new release v1 for acme-inc
Status: Finished processing events for release v1 of acme-inc`,
		},
		{
			"releases -a acme-inc",
			"v1    Dec 31  2014  Deploy remind101/acme-inc:latest (fake)",
		},
	})
}

func TestDeploy_CommitMessageRequired(t *testing.T) {
	pre := func(cli *CLI) {
		cli.Empire.MessagesRequired = true
	}

	runWithPre(t, []Command{
		{
			"deploy remind101/acme-inc",
			"error: A message is required for this action, please run again with '-m'.",
		},
	}, pre, true)

	runWithPre(t, []Command{
		{
			"deploy remind101/acme-inc -m commit",
			`Pulling repository remind101/acme-inc
345c7524bc96: Pulling image (latest) from remind101/acme-inc
345c7524bc96: Pulling image (latest) from remind101/acme-inc, endpoint: https://registry-1.docker.io/v1/
345c7524bc96: Pulling dependent layers
a1dd7097a8e8: Download complete
Status: Image is up to date for remind101/acme-inc:latest
Status: Created new release v1 for acme-inc
Status: Finished processing events for release v1 of acme-inc`,
		},
		{
			"releases -a acme-inc",
			"v1    Dec 31  2014  Deploy remind101/acme-inc:latest (fake: 'commit')",
		},
	}, pre, false)
}
