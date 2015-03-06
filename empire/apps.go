package empire

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
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

var (
	DockerRepo string = "docker"
	GitHubRepo string = "github"
)

// Repos represents the configured repos for an app.
type Repos struct {
	GitHub *Repo `json:"github" db:"github_repo"`
	Docker *Repo `json:"docker" db:"docker_repo"`
}

// App represents an app.
type App struct {
	Name AppName `json:"name" db:"name"`

	Repos // Any repos that this app is linked to.

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// Returns an error if the app isn't valid.
func (a *App) IsValid() error {
	if !NamePattern.Match([]byte(a.Name)) {
		return ErrInvalidName
	}

	return nil
}

// PreInsert implements a pre insert hook for the db interface
func (a *App) PreInsert(s gorp.SqlExecutor) error {
	a.CreatedAt = Now()
	return a.IsValid()
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
	AppsFindByRepo(string, Repo) (*App, error)
	AppsFindOrCreateByRepo(string, Repo) (*App, error)
}

type AppsService interface {
	AppsCreator
	AppsDestroyer
	AppsFinder
}

type appsService struct {
	DB
}

func (s *appsService) AppsCreate(app *App) (*App, error) {
	return AppsCreate(s.DB, app)
}

func (s *appsService) AppsDestroy(app *App) error {
	return AppsDestroy(s.DB, app)
}

func (s *appsService) AppsAll() ([]*App, error) {
	return AppsAll(s.DB)
}

func (s *appsService) AppsFind(name AppName) (*App, error) {
	return AppsFind(s.DB, name)
}

func (s *appsService) AppsFindByRepo(repoType string, repo Repo) (*App, error) {
	return AppsFindByRepo(s.DB, repoType, repo)
}

func (s *appsService) AppsFindOrCreateByRepo(repoType string, repo Repo) (*App, error) {
	return AppsFindOrCreateByRepo(s.DB, repoType, repo)
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
func AppsFindByRepo(db Queryier, repoType string, repo Repo) (*App, error) {
	return AppsFindBy(db, fmt.Sprintf("%s_repo", repoType), string(repo))
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
func AppsFindOrCreateByRepo(db DB, repoType string, repo Repo) (*App, error) {
	a, err := AppsFindByRepo(db, repoType, repo)
	if err != nil {
		return a, err
	}

	// If the app wasn't found, create a new up linked to this repo.
	if a == nil {
		n := NewAppNameFromRepo(repo)
		return AppsCreate(db, &App{
			Name: n,
			Repos: Repos{
				Docker: &repo,
			},
		})
	}

	return a, nil
}
