package empire

import (
	"reflect"
	"testing"
)

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
