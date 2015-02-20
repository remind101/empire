package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
)

func TestConfigsServiceApply(t *testing.T) {
	app := &apps.App{Name: "abcd"}
	s := NewConfigsService(nil)

	tests := []struct {
		in  configs.Vars
		out *configs.Config
	}{
		{
			configs.Vars{
				"RAILS_ENV": "production",
			},
			&configs.Config{
				Version: "20f3b833ad1f83353b1ae1d24ea6833693ce067c",
				App:     app,
				Vars: configs.Vars{
					"RAILS_ENV": "production",
				},
			},
		},
		{
			configs.Vars{
				"RAILS_ENV":    "production",
				"DATABASE_URL": "postgres://localhost",
			},
			&configs.Config{
				Version: "94a8e2be1e57b07526fee99473255a619563d551",
				App:     app,
				Vars: configs.Vars{
					"RAILS_ENV":    "production",
					"DATABASE_URL": "postgres://localhost",
				},
			},
		},
		{
			configs.Vars{
				"RAILS_ENV": "",
			},
			&configs.Config{
				Version: "aaa6f356d1507b0f5e14bb9adfddbea04d2569eb",
				App:     app,
				Vars: configs.Vars{
					"DATABASE_URL": "postgres://localhost",
				},
			},
		},
	}

	for _, tt := range tests {
		c, err := s.Apply(app, tt.in)

		if err != nil {
			t.Fatal(err)
		}

		if got, want := c, tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("want %q; got %q", want, got)
		}
	}
}
