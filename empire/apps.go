package empire

import (
	"database/sql"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"gopkg.in/gorp.v1"
)

var (
	// ErrInvalidName is used to indicate that the app name is not valid.
	ErrInvalidName = &ValidationError{
		errors.New("An app name must be alphanumeric and dashes only, 3-30 chars in length."),
	}
)

// NamePattern is a regex pattern that app names must conform to.
var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// NewAppNameFromRepo generates a new name from a Repo
//
//	remind101/r101-api => r101-api
func NewAppNameFromRepo(repo Repo) string {
	p := strings.Split(string(repo), "/")
	return p[len(p)-1]
}

// Repo types.
var (
	DockerRepo = "docker"
	GitHubRepo = "github"
)

// Repos represents the configured repos for an app.
type Repos struct {
	GitHub *Repo `json:"github" db:"github_repo"`
	Docker *Repo `json:"docker" db:"docker_repo"`
}

// Set sets the given repo type with the value.
func (r *Repos) Set(repoType string, value Repo) error {
	switch repoType {
	case GitHubRepo:
		r.GitHub = &value
	case DockerRepo:
		r.Docker = &value
	default:
		return fmt.Errorf("repo type not defined: %s", repoType)
	}

	return nil
}

// App represents an app.
type App struct {
	Name string `json:"name" db:"name"`

	Repos `json:"repos"` // Any repos that this app is linked to.

	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// IsValid returns an error if the app isn't valid.
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

type AppsUpdater interface {
	AppsUpdate(*App) (int64, error)
	AppsEnsureRepo(*App, string, Repo) error
}

type AppsDestroyer interface {
	AppsDestroy(*App) error
}

type AppsFinder interface {
	AppsAll() ([]*App, error)
	AppsFind(name string) (*App, error)
	AppsFindByRepo(string, Repo) (*App, error)
	AppsFindOrCreateByRepo(string, Repo) (*App, error)
}

type AppsService interface {
	AppsCreator
	AppsUpdater
	AppsDestroyer
	AppsFinder
}

type appsService struct {
	*db
	JobsService
}

func (s *appsService) AppsCreate(app *App) (*App, error) {
	return appsCreate(s.db, app)
}

func (s *appsService) AppsUpdate(app *App) (int64, error) {
	return appsUpdate(s.db, app)
}

func (s *appsService) AppsEnsureRepo(app *App, repoType string, repo Repo) error {
	return appsEnsureRepo(s.db, app, repoType, repo)
}

func (s *appsService) AppsDestroy(app *App) error {
	if err := appsDestroy(s.db, app); err != nil {
		return err
	}

	jobs, err := s.JobsList(JobsListQuery{App: app.Name})
	if err != nil {
		return err
	}

	return s.Unschedule(jobs...)
}

func (s *appsService) AppsAll() ([]*App, error) {
	return appsAll(s.db)
}

func (s *appsService) AppsFind(name string) (*App, error) {
	return appsFind(s.db, name)
}

func (s *appsService) AppsFindByRepo(repoType string, repo Repo) (*App, error) {
	return appsFindByRepo(s.db, repoType, repo)
}

func (s *appsService) AppsFindOrCreateByRepo(repoType string, repo Repo) (*App, error) {
	return appsFindOrCreateByRepo(s.db, repoType, repo)
}

// AppsCreate inserts the app into the database.
func appsCreate(db *db, app *App) (*App, error) {
	return app, db.Insert(app)
}

// AppsUpdate updates an app.
func appsUpdate(db *db, app *App) (int64, error) {
	return db.Update(app)
}

// AppsEnsureRepo will set the repo if it's not set.
func appsEnsureRepo(db *db, app *App, repoType string, repo Repo) error {
	switch repoType {
	case DockerRepo:
		if app.Repos.Docker != nil {
			return nil
		}
	case GitHubRepo:
		if app.Repos.GitHub != nil {
			return nil
		}
	}

	if err := app.Repos.Set(repoType, repo); err != nil {
		return err
	}

	_, err := appsUpdate(db, app)
	return err
}

// AppsDestroy destroys an app.
func appsDestroy(db *db, app *App) error {
	_, err := db.Delete(app)
	return err
}

// AppsAll returns all Apps.
func appsAll(db *db) ([]*App, error) {
	var apps []*App
	return apps, db.Select(&apps, `select * from apps order by name`)
}

// Finds an app by name.
func appsFind(db *db, name string) (*App, error) {
	return appsFindBy(db, "name", name)
}

// Finds an app by it's Repo field.
func appsFindByRepo(db *db, repoType string, repo Repo) (*App, error) {
	return appsFindBy(db, fmt.Sprintf("%s_repo", repoType), string(repo))
}

// AppsFindBy finds an app by a field.
func appsFindBy(db *db, field string, value interface{}) (*App, error) {
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
func appsFindOrCreateByRepo(db *db, repoType string, repo Repo) (*App, error) {
	a, err := appsFindByRepo(db, repoType, repo)
	if err != nil {
		return a, err
	}

	// If the app wasn't found, create a new app linked to this repo.
	if a != nil {
		return a, nil
	}

	n := NewAppNameFromRepo(repo)

	a, err = appsFind(db, n)
	if err != nil {
		return a, err
	}

	// If the app exists, update the repo value.
	if a != nil {
		return a, appsEnsureRepo(db, a, repoType, repo)
	}

	a = &App{Name: n}
	if err := a.Repos.Set(repoType, repo); err != nil {
		return a, err
	}

	return appsCreate(db, a)
}
