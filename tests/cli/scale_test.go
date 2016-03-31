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
			"scale -l -a acme-inc",
			"web=2:1X",
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

func TestScale_Constraints(t *testing.T) {
	run(t, []Command{
		DeployCommand("latest", "v1"),
		{
			"scale web=2:256:1GB -a acme-inc",
			"Scaled acme-inc to web=2:256:1.00gb.",
		},
		{
			"scale web=2:256:6GB -a acme-inc",
			"Scaled acme-inc to web=2:256:6.00gb.",
		},
		{
			"scale web=2:256:600GB -a acme-inc",
			"Scaled acme-inc to web=2:256:600.00gb.",
		},
	})
}
