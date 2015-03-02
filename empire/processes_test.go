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

type mockProcessesRepository struct {
	CreateFunc func(*Process) (*Process, error)
	UpdateFunc func(*Process) (int64, error)
	AllFunc    func(ReleaseID) (Formation, error)
}

func (r *mockProcessesRepository) Create(p *Process) (*Process, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(p)
	}

	return nil, nil
}

func (r *mockProcessesRepository) Update(p *Process) (int64, error) {
	if r.UpdateFunc != nil {
		return r.UpdateFunc(p)
	}

	return 0, nil
}

func (r *mockProcessesRepository) All(id ReleaseID) (Formation, error) {
	if r.AllFunc != nil {
		return r.All(id)
	}

	return nil, nil
}
