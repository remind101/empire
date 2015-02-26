package empire

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
)

var ErrInvalidName = errors.New("An app name must alphanumeric and dashes only, 3-30 chars in length.")

var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// AppName represents the unique name for an App.
type AppName string

// Scan implements the sql.Scanner interface.
func (n *AppName) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*n = AppName(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (n AppName) Value() (driver.Value, error) {
	return driver.Value(string(n)), nil
}

// NewNameFromRepo generates a new name from a Repo
//
//	remind101/r101-api => r101-api
func NewAppNameFromRepo(repo Repo) AppName {
	p := strings.Split(string(repo), "/")
	return AppName(p[len(p)-1])
}

// App represents an app.
type App struct {
	Name AppName `json:"name" db:"name"`

	// The associated GitHub/Docker repo.
	Repo Repo `json:"repo" db:"repo"`
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

// appsRepository is an implementation of the AppsRepository interface backed by
// a DB.
type appsRepository struct {
	DB
}

func NewAppsRepository(db DB) (AppsRepository, error) {
	return &appsRepository{db}, nil
}

func (r *appsRepository) Create(app *App) (*App, error) {
	return CreateApp(r.DB, app)
}

func (r *appsRepository) FindAll() ([]*App, error) {
	return AllApps(r.DB)
}

func (r *appsRepository) FindByName(name AppName) (*App, error) {
	return r.findBy("name", string(name))
}

func (r *appsRepository) FindByRepo(repo Repo) (*App, error) {
	return r.findBy("repo", string(repo))
}

func (r *appsRepository) findBy(field string, v interface{}) (*App, error) {
	var app App

	if err := r.SelectOne(&app, `select * from apps where `+field+` = $1 limit 1`, v); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &app, nil
}

// CreateApp inserts the app into the database.
func CreateApp(db Inserter, app *App) (*App, error) {
	return app, db.Insert(app)
}

// AllApps returns all Apps.
func AllApps(db Queryier) ([]*App, error) {
	var apps []*App
	return apps, db.Select(&apps, `select * from apps order by name`)
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
