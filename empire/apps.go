package empire

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"regexp"
	"strings"
	"time"

	"gopkg.in/gorp.v1"
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

	// The associated Docker repo.
	Repo Repo `json:"repo" db:"repo"`

	CreatedAt time.Time `json:"created_at" db:"created_at"`
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

// PreInsert implements a pre insert hook for the db interface
func (a *App) PreInsert(s gorp.SqlExecutor) error {
	a.CreatedAt = Now()
	return nil
}

// AppsRepository represents a repository for creating and finding Apps.
type AppsRepository interface {
	Create(*App) (*App, error)
	Destroy(*App) error
	FindAll() ([]*App, error)
	FindByName(AppName) (*App, error)
	FindByRepo(Repo) (*App, error)
}

// appsRepository is an implementation of the AppsRepository interface backed by
// a DB.
type appsRepository struct {
	DB
}

func (r *appsRepository) Create(app *App) (*App, error) {
	return CreateApp(r.DB, app)
}

func (r *appsRepository) Destroy(app *App) error {
	return DestroyApp(r.DB, app)
}

func (r *appsRepository) FindAll() ([]*App, error) {
	return AllApps(r.DB)
}

func (r *appsRepository) FindByName(name AppName) (*App, error) {
	return FindAppBy(r.DB, "name", string(name))
}

func (r *appsRepository) FindByRepo(repo Repo) (*App, error) {
	return FindAppBy(r.DB, "repo", string(repo))
}

// CreateApp inserts the app into the database.
func CreateApp(db Inserter, app *App) (*App, error) {
	return app, db.Insert(app)
}

// DestroyApp destroys an app.
func DestroyApp(db Deleter, app *App) error {
	_, err := db.Delete(app)
	return err
}

// AllApps returns all Apps.
func AllApps(db Queryier) ([]*App, error) {
	var apps []*App
	return apps, db.Select(&apps, `select * from apps order by name`)
}

// FindAppBy finds an app by a field.
func FindAppBy(db Queryier, field string, value interface{}) (*App, error) {
	var app App

	if err := findBy(db, &app, "apps", field, value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &app, nil
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
