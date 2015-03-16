package empire

import (
	"reflect"
	"testing"
)

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
					Type:     "web",
					Quantity: 1,
					Command:  "./bin/web",
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
					Type:     "web",
					Quantity: 1,
					Command:  "./bin/web",
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
					Type:     "worker",
					Quantity: 0,
					Command:  "sidekiq",
				},
			},
		},

		{
			f: Formation{
				"web": &Process{
					Type:     "web",
					Quantity: 5,
					Command:  "rackup",
				},
				"worker": &Process{
					Type:     "worker",
					Quantity: 2,
					Command:  "sidekiq",
				},
			},
			cm: CommandMap{
				"web": "./bin/web",
			},
			expected: Formation{
				"web": &Process{
					Type:     "web",
					Quantity: 5,
					Command:  "./bin/web",
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
