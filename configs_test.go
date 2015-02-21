package empire

import (
	"reflect"
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
)

func TestConfigsServiceApply(t *testing.T) {
	var pushed bool
	app := &apps.App{}

	r := &mockConfigsRepository{
		PushFunc: func(config *configs.Config) (*configs.Config, error) {
			pushed = true
			return config, nil
		},
	}
	s := &configsService{
		Repository: r,
	}

	config, err := s.Apply(app, configs.Vars{"RAILS_ENV": "production"})
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
	configs.Repository // Just to satisfy the interface.

	HeadFunc func(apps.Name) (*configs.Config, error)
	PushFunc func(*configs.Config) (*configs.Config, error)
}

func (r *mockConfigsRepository) Head(app apps.Name) (*configs.Config, error) {
	if r.HeadFunc != nil {
		return r.HeadFunc(app)
	}

	return nil, nil
}

func (r *mockConfigsRepository) Push(config *configs.Config) (*configs.Config, error) {
	if r.PushFunc != nil {
		return r.PushFunc(config)
	}

	return config, nil
}

type mockConfigsService struct {
	ConfigsService // Just to satisfy the interface.

	HeadFunc func(*apps.App) (*configs.Config, error)
}

func (s *mockConfigsService) Head(app *apps.App) (*configs.Config, error) {
	if s.HeadFunc != nil {
		return s.HeadFunc(app)
	}

	return nil, nil
}
