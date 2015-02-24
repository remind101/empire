package configs

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/apps"
)

func TestRepository(t *testing.T) {
	r := NewRepository()
	app := &apps.App{Name: "abcd"}

	c, _ := r.Push(&Config{App: app})
	if h, _ := r.Head(app.Name); !reflect.DeepEqual(c, h) {
		t.Fatalf("Head => %q; want %q", h, c)
	}

	if v, _ := r.Version(app.Name, c.Version); !reflect.DeepEqual(c, v) {
		t.Fatalf("Version(%s) => %q; want %q", c.Version, v, c)
	}
}

func TestMergeVars(t *testing.T) {
	// Old vars
	vars := Vars{
		"RAILS_ENV":    "production",
		"DATABASE_URL": "postgres://localhost",
	}

	tests := []struct {
		in  Vars
		out Vars
	}{
		// Removing a variable
		{
			Vars{
				"RAILS_ENV": "",
			},
			Vars{
				"DATABASE_URL": "postgres://localhost",
			},
		},

		// Updating a variable
		{
			Vars{
				"RAILS_ENV": "staging",
			},
			Vars{
				"RAILS_ENV":    "staging",
				"DATABASE_URL": "postgres://localhost",
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

func TestHash(t *testing.T) {
	tests := []struct {
		in  Vars
		out string
	}{
		// Simple
		{
			Vars{"RAILS_ENV": "production"},
			"20f3b833ad1f83353b1ae1d24ea6833693ce067c",
		},

		// More
		{
			Vars{"RAILS_ENV": "production", "FOO": "bar"},
			"e74293df4e696ca0247c3508456712a8541b826c",
		},

		// Swapped
		{
			Vars{"FOO": "bar", "RAILS_ENV": "production"},
			"e74293df4e696ca0247c3508456712a8541b826c",
		},
	}

	for _, tt := range tests {
		if got, want := hash(tt.in), tt.out; got != want {
			t.Errorf("hash(%q) => %s; want %s", tt.in, got, want)
		}
	}
}
