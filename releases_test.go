package empire

import (
	"testing"

	"github.com/remind101/empire/pkg/headerutil"
)

func TestReleasesQuery(t *testing.T) {
	var (
		app     = &App{ID: "1234"}
		version = 1
		sort    = "version"
		max     = 20
		order   = "desc"
	)

	rangeHeader := headerutil.Range{
		Sort:  &sort,
		Max:   &max,
		Order: &order,
	}

	tests := scopeTests{
		{ReleasesQuery{}, "ORDER BY version desc", []interface{}{}},
		{ReleasesQuery{Range: rangeHeader}, "ORDER BY version desc LIMIT 20", []interface{}{}},
		{ReleasesQuery{App: app, Range: rangeHeader}, "WHERE (app_id = $1) ORDER BY version desc LIMIT 20", []interface{}{"1234"}},
		{ReleasesQuery{Version: &version, Range: rangeHeader}, "WHERE (version = $1) ORDER BY version desc LIMIT 20", []interface{}{1}},
		{ReleasesQuery{App: app, Version: &version, Range: rangeHeader}, "WHERE (app_id = $1) AND (version = $2) ORDER BY version desc LIMIT 20", []interface{}{"1234", 1}},
	}

	tests.Run(t)
}
