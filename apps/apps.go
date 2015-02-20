package apps

import (
	"errors"
	"regexp"
	"strings"
	"sync"

	"github.com/remind101/empire/repos"
)

var ErrInvalidName = errors.New("An app name must alphanumeric and dashes only, 3-30 chars in length.")

var NamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]{2,30}$`)

// Name represents the unique name for an App.
type Name string

// NewNameFromRepo generates a new name from a Repo
//
//	remind101/r101-api => r101-api
func NewNameFromRepo(repo repos.Repo) Name {
	p := strings.Split(string(repo), "/")
	return Name(p[len(p)-1])
}

// App represents an app.
type App struct {
	Name Name `json:"name"`

	// The associated GitHub/Docker repo.
	Repo repos.Repo `json:"repo"`
}

// New validates the name of the new App then returns a new App instance. If the
// name is invalid, an error is retuend.
func New(name Name, repo repos.Repo) (*App, error) {
	if !NamePattern.Match([]byte(name)) {
		return nil, ErrInvalidName
	}

	return &App{
		Name: name,
		Repo: repo,
	}, nil
}

// NewFromRepo returns a new App initialized from the name of a Repo.
func NewFromRepo(repo repos.Repo) (*App, error) {
	name := NewNameFromRepo(repo)
	return New(name, repo)
}

// Repository represents a repository for creating and finding Apps.
type Repository interface {
	Create(*App) (*App, error)
	FindByName(Name) (*App, error)
	FindByRepo(repos.Repo) (*App, error)
}

func NewRepository() Repository {
	return newRepository()
}

type repository struct {
	id int

	sync.RWMutex
	apps []*App
}

func newRepository() *repository {
	return &repository{apps: make([]*App, 0)}
}

func (r *repository) Create(app *App) (*App, error) {
	r.Lock()
	defer r.Unlock()

	r.apps = append(r.apps, app)
	return app, nil
}

func (r *repository) FindByName(name Name) (*App, error) {
	r.RLock()
	defer r.RUnlock()

	for _, app := range r.apps {
		if app.Name == name {
			return app, nil
		}
	}

	return nil, nil
}

func (r *repository) FindByRepo(repo repos.Repo) (*App, error) {
	r.RLock()
	defer r.RUnlock()

	for _, app := range r.apps {
		if app.Repo == repo {
			return app, nil
		}
	}

	return nil, nil
}
