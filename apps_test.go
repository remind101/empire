package empire

import (
	"testing"
)

func TestNewApp(t *testing.T) {
	_, err := NewApp("", "")
	if err != ErrInvalidName {
		t.Error("An empty name should be invalid")
	}

	a, err := NewApp("api", "remind101/r101-api")
	if err != nil {
		t.Fatal(err)
	}

	if want, got := AppName("api"), a.Name; want != got {
		t.Errorf("a.Name => %s; want %s", got, want)

	}
}

func TestAppsServiceFindOrCreateByRepo(t *testing.T) {
	var created bool
	repo := Repo("remind101/r101-api")

	r := &mockAppsRepository{
		CreateFunc: func(app *App) (*App, error) {
			created = true
			return app, nil
		},
	}
	s := &appsService{
		AppsRepository: r,
	}

	app, err := s.FindOrCreateByRepo(repo)
	if err != nil {
		t.Fatal(err)
	}

	if got, want := created, true; got != want {
		t.Fatal("Expected the app to be created")
	}

	if got, want := app.Name, AppName("r101-api"); got != want {
		t.Fatal("Expected a name to be set")
	}

	if got, want := app.Repo, repo; got != want {
		t.Fatal("Expected the repo to be set")
	}
}

func TestAppsServiceFindOrCreateByRepoFound(t *testing.T) {
	r := &mockAppsRepository{
		CreateFunc: func(app *App) (*App, error) {
			t.Fatal("Expected Create to not be called")
			return nil, nil
		},
		FindByRepoFunc: func(repo Repo) (*App, error) {
			return &App{}, nil
		},
	}
	s := &appsService{
		AppsRepository: r,
	}

	if _, err := s.FindOrCreateByRepo("remind101/r101-api"); err != nil {
		t.Fatal(err)
	}
}

type mockAppsRepository struct {
	AppsRepository // Just to satisfy the interface.

	CreateFunc     func(*App) (*App, error)
	FindByRepoFunc func(Repo) (*App, error)
}

func (r *mockAppsRepository) Create(app *App) (*App, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(app)
	}

	return app, nil
}

func (r *mockAppsRepository) FindByRepo(repo Repo) (*App, error) {
	if r.FindByRepoFunc != nil {
		return r.FindByRepoFunc(repo)
	}

	return nil, nil
}

type mockAppsService struct {
	mockAppsRepository

	FindOrCreateByRepoFunc func(repo Repo) (*App, error)
}

func (s *mockAppsService) FindOrCreateByRepo(repo Repo) (*App, error) {
	if s.FindOrCreateByRepoFunc != nil {
		return s.FindOrCreateByRepoFunc(repo)
	}

	return nil, nil
}
