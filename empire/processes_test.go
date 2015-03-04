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

type mockProcessesService struct {
	ProcessesCreateFunc func(*Process) (*Process, error)
	ProcessesUpdateFunc func(*Process) (int64, error)
	ProcessesAllFunc    func(*Release) (Formation, error)
}

func (r *mockProcessesService) ProcessesCreate(p *Process) (*Process, error) {
	if r.ProcessesCreateFunc != nil {
		return r.ProcessesCreateFunc(p)
	}

	return nil, nil
}

func (r *mockProcessesService) ProcessesUpdate(p *Process) (int64, error) {
	if r.ProcessesUpdateFunc != nil {
		return r.ProcessesUpdateFunc(p)
	}

	return 0, nil
}

func (r *mockProcessesService) ProcessesAll(release *Release) (Formation, error) {
	if r.ProcessesAllFunc != nil {
		return r.ProcessesAll(release)
	}

	return nil, nil
}
