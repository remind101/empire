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
					Quantity: 0,
					Command:  "sidekiq",
				},
			},
		},

		{
			f: Formation{
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
			expected: Formation{
				"web": &Process{
					Quantity: 5,
					Command:  "./bin/web",
				},
			},
		},
	}

	for _, tt := range tests {
		f := NewFormation(tt.f, tt.cm)

		if got, want := f, tt.expected; !reflect.DeepEqual(got, want) {
			t.Fatalf("processes => %v; want %v", got, want)
		}
	}
}

type mockProcessesRepository struct {
	CreateFunc func(ProcessType, *Process) (ProcessType, *Process, error)
	AllFunc    func(ReleaseID) (Formation, error)
}

func (r *mockProcessesRepository) Create(t ProcessType, p *Process) (ProcessType, *Process, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(t, p)
	}

	return "", nil, nil
}

func (r *mockProcessesRepository) All(id ReleaseID) (Formation, error) {
	if r.AllFunc != nil {
		return r.All(id)
	}

	return nil, nil
}
