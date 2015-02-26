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

func TestConfigsServiceApply(t *testing.T) {
	var pushed bool
	app := &App{}

	r := &mockConfigsRepository{
		PushFunc: func(config *Config) (*Config, error) {
			pushed = true
			return config, nil
		},
	}
	s := &configsService{
		Repository: r,
	}

	config, err := s.Apply(app, Vars{"RAILS_ENV": "production"})
	if err != nil {
		t.Fatal(err)
	}

	if got, want := pushed, true; got != want {
		t.Fatal("Expected the config to be pushed")
	}

	if got, want := config.App, app; !reflect.DeepEqual(got, want) {
		t.Fatal("Expected App to be set on config")
	}
}

type mockConfigsRepository struct {
	ConfigsRepository // Just to satisfy the interface.

	HeadFunc func(AppName) (*Config, error)
	PushFunc func(*Config) (*Config, error)
}

func (r *mockConfigsRepository) Head(app AppName) (*Config, error) {
	if r.HeadFunc != nil {
		return r.HeadFunc(app)
	}

	return nil, nil
}

func (r *mockConfigsRepository) Push(config *Config) (*Config, error) {
	if r.PushFunc != nil {
		return r.PushFunc(config)
	}

	return config, nil
}

type mockConfigsService struct {
	ConfigsService // Just to satisfy the interface.

	HeadFunc func(*App) (*Config, error)
}

func (s *mockConfigsService) Head(app *App) (*Config, error) {
	if s.HeadFunc != nil {
		return s.HeadFunc(app)
	}

	return nil, nil
}
