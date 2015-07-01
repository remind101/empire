package empire

import (
	"encoding/json"
	"fmt"
	"reflect"
	"testing"

	. "github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/constraints"
)

func TestProcessesQuery(t *testing.T) {
	release := &Release{ID: "1234"}

	tests := scopeTests{
		{ProcessesQuery{}, "", []interface{}{}},
		{ProcessesQuery{Release: release}, "WHERE (release_id = $1)", []interface{}{release.ID}},
	}

	tests.Run(t)
}

func TestNewFormation(t *testing.T) {
	tests := []struct {
		f  Formation
		cm CommandMap

		expected Formation
	}{

		{
			f: nil,
			cm: CommandMap{
				"web": "./bin/web",
			},
			expected: Formation{
				"web": &Process{
					Type:        "web",
					Quantity:    1,
					Command:     "./bin/web",
					Constraints: NamedConstraints["1X"],
				},
			},
		},

		{
			f: Formation{},
			cm: CommandMap{
				"web": "./bin/web",
			},
			expected: Formation{
				"web": &Process{
					Type:        "web",
					Quantity:    1,
					Command:     "./bin/web",
					Constraints: NamedConstraints["1X"],
				},
			},
		},

		{
			f: Formation{},
			cm: CommandMap{
				"worker": "sidekiq",
			},
			expected: Formation{
				"worker": &Process{
					Type:        "worker",
					Quantity:    0,
					Command:     "sidekiq",
					Constraints: NamedConstraints["1X"],
				},
			},
		},

		{
			f: Formation{
				"web": &Process{
					Type:        "web",
					Quantity:    5,
					Command:     "rackup",
					Constraints: NamedConstraints["1X"],
				},
				"worker": &Process{
					Type:        "worker",
					Quantity:    2,
					Command:     "sidekiq",
					Constraints: NamedConstraints["1X"],
				},
			},
			cm: CommandMap{
				"web": "./bin/web",
			},
			expected: Formation{
				"web": &Process{
					Type:        "web",
					Quantity:    5,
					Command:     "./bin/web",
					Constraints: NamedConstraints["1X"],
				},
			},
		},
	}

	for i, tt := range tests {
		f := NewFormation(tt.f, tt.cm)

		if got, want := f, tt.expected; !reflect.DeepEqual(got, want) {
			t.Fatalf("%d processes => %v; want %v", i, got, want)
		}
	}
}

func TestConstraints_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		in  string
		out Constraints
		err error
	}{
		{"512:1KB", Constraints{512, 1024}, nil},
		{"1025:1KB", Constraints{}, constraints.ErrInvalidCPUShare},

		{"1024", Constraints{}, constraints.ErrInvalidConstraint},
	}

	for i, tt := range tests {
		var c Constraints

		err := json.Unmarshal([]byte(fmt.Sprintf(`"%s"`, tt.in)), &c)
		if err != tt.err {
			t.Fatalf("#%d: err => %v; want %v", i, err, tt.err)
		}

		if tt.err != nil {
			continue
		}

		if got, want := c, tt.out; !reflect.DeepEqual(got, want) {
			t.Fatalf("#%d: Constraints => %v; want %v", i, got, want)
		}
	}
}

func TestConstraints_String(t *testing.T) {
	tests := []struct {
		in  Constraints
		out string
	}{
		// Named constraints
		{Constraints1X, "1X"},
		{Constraints2X, "2X"},
		{ConstraintsPX, "PX"},

		{Constraints{100, constraints.Memory(1 * MB)}, "100:1.00mb"},
	}

	for _, tt := range tests {
		out := tt.in.String()

		if got, want := out, tt.out; got != want {
			t.Fatalf(".String() => %s; want %s", got, want)
		}
	}
}
