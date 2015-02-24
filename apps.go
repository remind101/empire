package empire

import (
	"errors"
	"regexp"
	"strings"

	"github.com/remind101/empire/stores"
)

var ErrInvalidName = errors.New("An app name must alphanumeric and dashes only, 3-30 chars in length.")

var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// AppName represents the unique name for an App.
type AppName string

// NewNameFromRepo generates a new name from a Repo
//
//	remind101/r101-api => r101-api
func NewAppNameFromRepo(repo Repo) AppName {
	p := strings.Split(string(repo), "/")
	return AppName(p[len(p)-1])
}

// App represents an app.
type App struct {
	Name AppName `json:"name"`

	// The associated GitHub/Docker repo.
	Repo Repo `json:"repo"`
}

// NewApp validates the name of the new App then returns a new App instance. If the
// name is invalid, an error is retuend.
func NewApp(name AppName, repo Repo) (*App, error) {
	if !NamePattern.Match([]byte(name)) {
		return nil, ErrInvalidName
	}

	return &App{
		Name: name,
		Repo: repo,
	}, nil
}

// NewAppFromRepo returns a new App initialized from the name of a Repo.
func NewAppFromRepo(repo Repo) (*App, error) {
	name := NewAppNameFromRepo(repo)
	return NewApp(name, repo)
}

// AppsRepository represents a repository for creating and finding Apps.
type AppsRepository interface {
	Create(*App) (*App, error)
	FindByName(AppName) (*App, error)
	FindByRepo(Repo) (*App, error)
}

type appsRepository struct {
	s stores.Store
}

func NewAppsRepository() AppsRepository {
	return &appsRepository{stores.NewMemStore()}
}

func NewEtcdAppsRepository(ns string) (AppsRepository, error) {
	s, err := stores.NewEtcdStore(ns)
	if err != nil {
		return nil, err
	}
	return &appsRepository{s}, nil
}

func (r *appsRepository) Create(app *App) (*App, error) {
	err := r.s.Set(string(app.Name), app)
	return app, err
}

func (r *appsRepository) FindByName(name AppName) (*App, error) {
	apps := make([]*App, 0)
	if err := r.s.List("", &apps); err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.Name == name {
			return app, nil
		}
	}

	return nil, nil
}

func (r *appsRepository) FindByRepo(repo Repo) (*App, error) {
	apps := make([]*App, 0)
	if err := r.s.List("", &apps); err != nil {
		return nil, err
	}

	for _, app := range apps {
		if app.Repo == repo {
			return app, nil
		}
	}

	return nil, nil
}

// AppsService represents a service for interacting with Apps.
type AppsService interface {
	AppsRepository

	// FindOrCreateByRepo attempts to find an app by a repo name, or creates
	// a new app if it's not found.
	FindOrCreateByRepo(Repo) (*App, error)
}

// appsService is a base implementation of the AppsService interface.
type appsService struct {
	AppsRepository
}

// NewAppsService returns a new Service instance.
func NewAppsService(options Options) (AppsService, error) {
	return &appsService{
		AppsRepository: NewAppsRepository(),
	}, nil
}

func (s *appsService) FindOrCreateByRepo(repo Repo) (*App, error) {
	a, err := s.AppsRepository.FindByRepo(repo)
	if err != nil {
		return a, err
	}

	// If the app wasn't found, create a new up linked to this repo.
	if a == nil {
		a, err := NewAppFromRepo(repo)
		if err != nil {
			return a, err
		}
		return s.AppsRepository.Create(a)
	}

	return a, nil
}
