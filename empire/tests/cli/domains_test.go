package cli_test

import "testing"

func TestDomains(t *testing.T) {
	run(t, []Command{
		{
			"create acme-inc",
			"Created acme-inc.",
		},
		{
			"domains -a acme-inc",
			"",
		},
		{
			"domain-add example.com -a acme-inc",
			"Added example.com to acme-inc.",
		},
		{
			"domains -a acme-inc",
			"example.com",
		},
		{
			"domain-remove example.com -a acme-inc",
			"Removed example.com from acme-inc.",
		},
		{
			"domains -a acme-inc",
			"",
		},
	})

}
