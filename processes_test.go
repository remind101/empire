package empire

import (
	"reflect"
	"testing"
)

func TestNewProcessMap(t *testing.T) {
	tests := []struct {
		pm ProcessMap
		cm CommandMap

		expected ProcessMap
	}{
		{
			pm: ProcessMap{},
			cm: CommandMap{
				"web": "./bin/web",
			},
			expected: ProcessMap{
				"web": &Process{
					Quantity: 1,
					Command:  "./bin/web",
				},
			},
		},

		{
			pm: ProcessMap{},
			cm: CommandMap{
				"worker": "sidekiq",
			},
			expected: ProcessMap{
				"worker": &Process{
					Quantity: 0,
					Command:  "sidekiq",
				},
			},
		},

		{
			pm: ProcessMap{
				"web": &Process{
					Quantity: 5,
					Command:  "rackup",
				},
				"worker": &Process{
					Quantity: 2,
					Command:  "sidekiq",
				},
			},
			cm: CommandMap{
				"web": "./bin/web",
			},
			expected: ProcessMap{
				"web": &Process{
					Quantity: 5,
					Command:  "./bin/web",
				},
			},
		},
	}

	for _, tt := range tests {
		pm := NewProcessMap(tt.pm, tt.cm)

		if got, want := pm, tt.expected; !reflect.DeepEqual(got, want) {
			t.Fatalf("processes => %v; want %v", got, want)
		}
	}
}
