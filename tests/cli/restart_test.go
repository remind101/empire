package cli_test

import (
	"testing"
	"time"
)

func testRestart(t *testing.T) {
	now(time.Now().AddDate(0, 0, -5))
	defer resetNow()

	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		DeployCommand("latest", "v2"),
		{
			"scale web=2 -a acme-inc",
			"Scaled acme-inc to web=2:1X.",
		},
		//{
		//"dynos -a acme-inc",
		//`v1.web.1  i-aa111aa1  1X  running   5d  "./bin/web"
		//v1.web.2  i-aa111aa1  1X  running   5d  "./bin/web"`,
		//},
		{
			"restart -a acme-inc",
			"Restarted all dynos for acme-inc.",
		},
		{
			"restart 1 -a acme-inc",
			"Restarted 1 dynos for acme-inc.",
		},
	})
}
