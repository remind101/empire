package empire

import (
	"database/sql"
	"errors"
	"regexp"
	"strings"
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
	FindAll() ([]*App, error)
	FindByName(AppName) (*App, error)
	FindByRepo(Repo) (*App, error)
}

// dbApp represents the db representation of an app.
type dbApp struct {
	Name string `db:"name"`
	Repo string `db:"repo"`
}

// appsRepository is an implementation of the AppsRepository interface backed by
// a DB.
type appsRepository struct {
	DB
}

func NewAppsRepository(db DB) (AppsRepository, error) {
	return &appsRepository{db}, nil
}

func (r *appsRepository) Create(app *App) (*App, error) {
	a := &dbApp{
		Name: string(app.Name),
		Repo: string(app.Repo),
	}

	if err := r.DB.Insert(a); err != nil {
		return app, err
	}

	return toApp(a, app), nil
}

func (r *appsRepository) FindAll() ([]*App, error) {
	var dbapps []*dbApp
	if err := r.Select(&dbapps, `select * from apps order by name`); err != nil {
		return nil, err
	}

	apps := make([]*App, len(dbapps))
	for i, a := range dbapps {
		apps[i] = toApp(a, nil)
	}

	return apps, nil
}

func (r *appsRepository) FindByName(name AppName) (*App, error) {
	return r.findBy("name", string(name))
}

func (r *appsRepository) FindByRepo(repo Repo) (*App, error) {
	return r.findBy("repo", string(repo))
}

func (r *appsRepository) findBy(field string, v interface{}) (*App, error) {
	var a dbApp

	if err := r.SelectOne(&a, `select * from apps where `+field+` = $1 limit 1`, v); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return toApp(&a, nil), nil
}

// toApp maps a dbApp to an App.
func toApp(a *dbApp, app *App) *App {
	if app == nil {
		app = &App{}
	}

	app.Name = AppName(a.Name)
	app.Repo = Repo(a.Repo)

	return app
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
func NewAppsService(r AppsRepository) (AppsService, error) {
	return &appsService{
		AppsRepository: r,
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
