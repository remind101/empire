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
	AppsCreate(*App) (*App, error)
}

type AppsDestroyer interface {
	AppsDestroy(*App) error
}

type AppsFinder interface {
	AppsAll() ([]*App, error)
	AppsFind(AppName) (*App, error)
	AppsFindByRepo(Repo) (*App, error)
	AppsFindOrCreateByRepo(Repo) (*App, error)
}

type AppsService interface {
	AppsCreator
	AppsDestroyer
	AppsFinder
}

type appsService struct {
	DB
	JobsService
}

func (s *appsService) AppsCreate(app *App) (*App, error) {
	return AppsCreate(s.DB, app)
}

func (s *appsService) AppsDestroy(app *App) error {
	if err := AppsDestroy(s.DB, app); err != nil {
		return err
	}

	jobs, err := s.JobsList(JobsListQuery{App: app.Name})
	if err != nil {
		return err
	}

	if err := s.Unschedule(jobs...); err != nil {
		return err
	}

	return nil
}

func (s *appsService) AppsAll() ([]*App, error) {
	return AppsAll(s.DB)
}

func (s *appsService) AppsFind(name AppName) (*App, error) {
	return AppsFind(s.DB, name)
}

func (s *appsService) AppsFindByRepo(repo Repo) (*App, error) {
	return AppsFindByRepo(s.DB, repo)
}

func (s *appsService) AppsFindOrCreateByRepo(repo Repo) (*App, error) {
	return AppsFindOrCreateByRepo(s.DB, repo)
}

// AppsCreate inserts the app into the database.
func AppsCreate(db Inserter, app *App) (*App, error) {
	return app, db.Insert(app)
}

// AppsDestroy destroys an app.
func AppsDestroy(db Deleter, app *App) error {
	_, err := db.Delete(app)
	return err
}

// AppsAll returns all Apps.
func AppsAll(db Queryier) ([]*App, error) {
	var apps []*App
	return apps, db.Select(&apps, `select * from apps order by name`)
}

// Finds an app by name.
func AppsFind(db Queryier, name AppName) (*App, error) {
	return AppsFindBy(db, "name", string(name))
}

// Finds an app by it's Repo field.
func AppsFindByRepo(db Queryier, repo Repo) (*App, error) {
	return AppsFindBy(db, "repo", string(repo))
}

// AppsFindBy finds an app by a field.
func AppsFindBy(db Queryier, field string, value interface{}) (*App, error) {
	var app App

	if err := findBy(db, &app, "apps", field, value); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}

		return nil, err
	}

	return &app, nil
}

// AppsFindOrCreateByRepo first attempts to find an app by repo, falling back to
// creating a new app.
func AppsFindOrCreateByRepo(db DB, repo Repo) (*App, error) {
	a, err := AppsFindByRepo(db, repo)
	if err != nil {
		return a, err
	}

	// If the app wasn't found, create a new up linked to this repo.
	if a == nil {
		a, err := NewAppFromRepo(repo)
		if err != nil {
			return a, err
		}
		return AppsCreate(db, a)
	}

	return a, nil
}
