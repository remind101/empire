package empire

import (
	"testing"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/repos"
)

func TestAppsServiceFindOrCreateByRepo(t *testing.T) {
	var created bool
	repo := repos.Repo("remind101/r101-api")

	r := &mockAppsRepository{
		CreateFunc: func(app *apps.App) (*apps.App, error) {
			created = true
			return app, nil
		},
	}
	s := &appsService{
		Repository: r,
	}

	app, err := s.FindOrCreateByRepo(repo)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := created, true; got != want {
		t.Fatal("Expected the app to be created")
	}

	if got, want := app.Name, apps.Name("r101-api"); got != want {
		t.Fatal("Expected a name to be set")
	}

	if got, want := app.Repo, repo; got != want {
		t.Fatal("Expected the repo to be set")
	}
}

func TestAppsServiceFindOrCreateByRepoFound(t *testing.T) {
	r := &mockAppsRepository{
		CreateFunc: func(app *apps.App) (*apps.App, error) {
			t.Fatal("Expected Create to not be called")
			return nil, nil
		},
		FindByRepoFunc: func(repo repos.Repo) (*apps.App, error) {
			return &apps.App{}, nil
		},
	}
	s := &appsService{
		Repository: r,
	}

	if _, err := s.FindOrCreateByRepo("remind101/r101-api"); err != nil {
		t.Fatal(err)
	}
}

type mockAppsRepository struct {
	apps.Repository // Just to satisfy the interface.

	CreateFunc     func(*apps.App) (*apps.App, error)
	FindByRepoFunc func(repos.Repo) (*apps.App, error)
}

func (r *mockAppsRepository) Create(app *apps.App) (*apps.App, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(app)
	}

	return app, nil
}

func (r *mockAppsRepository) FindByRepo(repo repos.Repo) (*apps.App, error) {
	if r.FindByRepoFunc != nil {
		return r.FindByRepoFunc(repo)
	}

	return nil, nil
}

type mockAppsService struct {
	mockAppsRepository

	FindOrCreateByRepoFunc func(repo repos.Repo) (*apps.App, error)
}

func (s *mockAppsService) FindOrCreateByRepo(repo repos.Repo) (*apps.App, error) {
	if s.FindOrCreateByRepoFunc != nil {
		return s.FindOrCreateByRepoFunc(repo)
	}

	return nil, nil
}
