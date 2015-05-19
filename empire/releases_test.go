package empire

import "testing"

func TestReleasesQuery(t *testing.T) {
	app := &App{ID: "1234"}
	version := 1

	tests := []scopeTest{
		{ReleasesQuery{}, "ORDER BY version desc", []interface{}{}},
		{ReleasesQuery{App: app}, "WHERE (app_id = $1) ORDER BY version desc", []interface{}{"1234"}},
		{ReleasesQuery{Version: &version}, "WHERE (version = $1) ORDER BY version desc", []interface{}{1}},
		{ReleasesQuery{App: app, Version: &version}, "WHERE (app_id = $1) AND (version = $2) ORDER BY version desc", []interface{}{"1234", 1}},
	}

	for _, tt := range tests {
		assertScopeSql(t, tt.scope, tt.sql, tt.vars...)
	}
}
