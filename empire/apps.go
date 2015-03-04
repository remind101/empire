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

type AppsCreator interface {
	Create(*App) (*App, error)
}

type AppsDestroyer interface {
	Destroy(*App) error
}

type AppsFinder interface {
	All() ([]*App, error)
	Find(AppName) (*App, error)
	FindByRepo(Repo) (*App, error)
	FindOrCreateByRepo(Repo) (*App, error)
}

type AppsService interface {
	AppsCreator
	AppsDestroyer
	AppsFinder
}

type appsService struct {
	DB
}

func (s *appsService) Create(app *App) (*App, error) {
	return CreateApp(s.DB, app)
}

func (s *appsService) Destroy(app *App) error {
	return DestroyApp(s.DB, app)
}

func (s *appsService) All() ([]*App, error) {
	return AllApps(s.DB)
}

func (s *appsService) Find(name AppName) (*App, error) {
	return FindApp(s.DB, name)
}

func (s *appsService) FindByRepo(repo Repo) (*App, error) {
	return FindAppByRepo(s.DB, repo)
}

func (s *appsService) FindOrCreateByRepo(repo Repo) (*App, error) {
	return FindOrCreateAppByRepo(s.DB, repo)
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

// Finds an app by name.
func FindApp(db Queryier, name AppName) (*App, error) {
	return FindAppBy(db, "name", string(name))
}

// Finds an app by it's Repo field.
func FindAppByRepo(db Queryier, repo Repo) (*App, error) {
	return FindAppBy(db, "repo", string(repo))
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

func FindOrCreateAppByRepo(db DB, repo Repo) (*App, error) {
	a, err := FindAppByRepo(db, repo)
	if err != nil {
		return a, err
	}

	// If the app wasn't found, create a new up linked to this repo.
	if a == nil {
		a, err := NewAppFromRepo(repo)
		if err != nil {
			return a, err
		}
		return CreateApp(db, a)
	}

	return a, nil
}
