package cli_test

import (
	"testing"
	"time"
)

func TestMaintenance(t *testing.T) {
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
			"rake=0:1X scheduled=0:1X web=2:1X worker=0:1X",
		},
		{
			"ps -a acme-inc",
			`v1.web.1  i-aa111aa1  1X  running   5d  "./bin/web"
v1.web.2  i-aa111aa1  1X  running   5d  "./bin/web"`,
		},
		{
			"maintenance-enable -a acme-inc",
			"Enabled maintenance mode on acme-inc.",
		},
		{
			"ps -a acme-inc",
			``,
		},
		{
			"scale -l -a acme-inc",
			"rake=0:1X scheduled=0:1X web=2:1X worker=0:1X",
		},
		{
			"maintenance -a acme-inc",
			"enabled",
		},
		{
			"maintenance-disable -a acme-inc",
			"Disabled maintenance mode on acme-inc.",
		},
		{
			"ps -a acme-inc",
			`v1.web.1  i-aa111aa1  1X  running   5d  "./bin/web"
v1.web.2  i-aa111aa1  1X  running   5d  "./bin/web"`,
		},
	})
}
