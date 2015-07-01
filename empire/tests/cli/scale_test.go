package cli_test

import (
	"testing"
	"time"
)

func TestScale(t *testing.T) {
	now(time.Now().AddDate(0, 0, -5))
	defer resetNow()

	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"scale web=2 -a acme-inc",
			"Scaled acme-inc to web=2:1X.",
		},
		{
			"dynos -a acme-inc",
			`v1.web.1  1X  running   5d  "./bin/web"
v1.web.2  1X  running   5d  "./bin/web"`,
		},

		{
			"scale web=1 -a acme-inc",
			"Scaled acme-inc to web=1:1X.",
		},
		{
			"dynos -a acme-inc",
			"v1.web.1  1X  running   5d  \"./bin/web\"",
		},
	})
}
