package empire

import "testing"

func TestDomainsQuery(t *testing.T) {
	hostname := "acme-inc.classchirp.com"
	app := &App{ID: "1234"}

	tests := scopeTests{
		{DomainsQuery{}, "", []interface{}{}},
		{DomainsQuery{Hostname: &hostname}, "WHERE (hostname = $1)", []interface{}{hostname}},
		{DomainsQuery{App: app}, "WHERE (app_id = $1)", []interface{}{app.ID}},
	}

	tests.Run(t)
}
