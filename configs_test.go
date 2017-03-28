package empire

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/stretchr/testify/assert"
)

func TestConfigsQuery(t *testing.T) {
	id := "1234"
	app := &App{ID: "4321"}

	tests := scopeTests{
		{ConfigsQuery{}, "", []interface{}{}},
		{ConfigsQuery{ID: &id}, "WHERE (id = $1)", []interface{}{id}},
		{ConfigsQuery{App: app}, "WHERE (app_id = $1)", []interface{}{app.ID}},
	}

	tests.Run(t)
}

func TestMergeVars(t *testing.T) {
	var (
		PRODUCTION   = "production"
		STAGING      = "staging"
		EMPTY        = ""
		DATABASE_URL = "postgres://localhost"
	)

	// Old vars
	vars := Vars{
		"RAILS_ENV":    &PRODUCTION,
		"DATABASE_URL": &DATABASE_URL,
	}

	tests := []struct {
		in  Vars
		out Vars
	}{
		// Removing a variable
		{
			Vars{
				"RAILS_ENV": nil,
			},
			Vars{
				"DATABASE_URL": &DATABASE_URL,
			},
		},

		// Setting an empty variable
		{
			Vars{
				"RAILS_ENV": &EMPTY,
			},
			Vars{
				"RAILS_ENV":    &EMPTY,
				"DATABASE_URL": &DATABASE_URL,
			},
		},

		// Updating a variable
		{
			Vars{
				"RAILS_ENV": &STAGING,
			},
			Vars{
				"RAILS_ENV":    &STAGING,
				"DATABASE_URL": &DATABASE_URL,
			},
		},
	}

	for _, tt := range tests {
		v := mergeVars(vars, tt.in)

		if got, want := v, tt.out; !reflect.DeepEqual(got, want) {
			t.Errorf("mergeVars => want %v; got %v", want, got)
		}
	}
}

func TestDiffVars(t *testing.T) {
	tests := []struct {
		a, b     Vars
		expected *VarsDiff
	}{
		{Vars{"RAILS_ENV": aws.String("production")}, Vars{}, &VarsDiff{Added: []string{"RAILS_ENV"}}},
		{Vars{}, Vars{"RAILS_ENV": aws.String("production")}, &VarsDiff{Removed: []string{"RAILS_ENV"}}},
		{Vars{"RAILS_ENV": aws.String("staging")}, Vars{"RAILS_ENV": aws.String("production")}, &VarsDiff{Changed: []string{"RAILS_ENV"}}},
		{Vars{"COOKIE_SECRET": aws.String("secret")}, Vars{"RAILS_ENV": aws.String("production")}, &VarsDiff{Removed: []string{"RAILS_ENV"}, Added: []string{"COOKIE_SECRET"}}},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			diff := DiffVars(tt.a, tt.b)
			assert.Equal(t, tt.expected, diff)
		})
	}
}

func TestReleaseDesc(t *testing.T) {
	configVal := "test"

	tests := []struct {
		diff *VarsDiff
		opts SetOpts
		out  string
	}{
		{
			&VarsDiff{
				Added: []string{"FOO"},
			},
			SetOpts{
				User: &User{Name: "fake"},
				Vars: Vars{"FOO": &configVal},
			},
			"Added (FOO) config var (fake)",
		},
		{
			&VarsDiff{
				Changed: []string{"FOO"},
				Added:   []string{"BAR"},
			},
			SetOpts{
				User:    &User{Name: "fake"},
				Vars:    Vars{"FOO": &configVal, "BAR": &configVal},
				Message: "important things",
			},
			"Added (BAR) Changed (FOO) config vars (fake: 'important things')",
		},
		{
			&VarsDiff{
				Changed: []string{"FOO"},
				Removed: []string{"BAR"},
			},
			SetOpts{
				User:    &User{Name: "fake"},
				Vars:    Vars{"FOO": &configVal, "BAR": nil},
				Message: "important things",
			},
			"Changed (FOO) Removed (BAR) config vars (fake: 'important things')",
		},
		{
			&VarsDiff{
				Removed: []string{"FOO"},
			},
			SetOpts{
				User: &User{Name: "fake"},
				Vars: Vars{"FOO": nil},
			},
			"Removed (FOO) config var (fake)",
		},
		{
			&VarsDiff{
				Removed: []string{"FOO", "BAR"},
			},
			SetOpts{
				User:    &User{Name: "fake"},
				Vars:    Vars{"FOO": nil, "BAR": nil},
				Message: "important things",
			},
			"Removed (FOO, BAR) config vars (fake: 'important things')",
		},
		{
			&VarsDiff{},
			SetOpts{
				User:    &User{Name: "fake"},
				Vars:    Vars{"FOO": &configVal},
				Message: "important things",
			},
			"Made no changes to config vars (fake: 'important things')",
		},
	}

	for _, tt := range tests {
		d := configsApplyReleaseDesc(tt.diff, tt.opts)

		if got, want := d, tt.out; got != want {
			t.Errorf("configsApplyReleaseDesc => want %v; got %v", want, got)
		}
	}
}
