package empire

import (
	"reflect"
	"testing"
)

func TestMergeVars(t *testing.T) {
	var (
		PRODUCTION   = "production"
		STAGING      = "staging"
		EMPTY        = ""
		DATABASE_URL = "postgres://localhost"
	)

	// Old vars
	vars := map[string]string{
		"RAILS_ENV":    PRODUCTION,
		"DATABASE_URL": DATABASE_URL,
	}

	tests := []struct {
		changes Vars
		out     map[string]string
	}{
		// Removing a variable
		{
			Vars{
				"RAILS_ENV": nil,
			},
			map[string]string{
				"DATABASE_URL": DATABASE_URL,
			},
		},

		// Setting an empty variable
		{
			Vars{
				"RAILS_ENV": &EMPTY,
			},
			map[string]string{
				"RAILS_ENV":    EMPTY,
				"DATABASE_URL": DATABASE_URL,
			},
		},

		// Updating a variable
		{
			Vars{
				"RAILS_ENV": &STAGING,
			},
			map[string]string{
				"RAILS_ENV":    STAGING,
				"DATABASE_URL": DATABASE_URL,
			},
		},
	}

	for _, tt := range tests {
		v := mergeVars(vars, tt.changes)

		if got, want := v, tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("mergeVars => want %v; got %v", want, got)
		}
	}
}

func TestReleaseDesc(t *testing.T) {
	configVal := "test"

	tests := []struct {
		in  SetOpts
		out string
	}{
		{
			SetOpts{
				User: &User{Name: "fake"},
				Vars: Vars{"FOO": &configVal},
			},
			"Set FOO config var (fake)",
		},
		{
			SetOpts{
				User:    &User{Name: "fake"},
				Vars:    Vars{"FOO": &configVal, "BAR": &configVal},
				Message: "important things",
			},
			"Set BAR, FOO config vars (fake: 'important things')",
		},
		{
			SetOpts{
				User: &User{Name: "fake"},
				Vars: Vars{"FOO": nil},
			},
			"Unset FOO config var (fake)",
		},
		{
			SetOpts{
				User:    &User{Name: "fake"},
				Vars:    Vars{"FOO": nil, "BAR": nil},
				Message: "important things",
			},
			"Unset BAR, FOO config vars (fake: 'important things')",
		},
	}

	for _, tt := range tests {
		d := configsApplyReleaseDesc(tt.in)

		if got, want := d, tt.out; got != want {
			t.Errorf("configsApplyReleaseDesc => want %v; got %v", want, got)
		}
	}
}
