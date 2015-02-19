package apps

import (
	"errors"
	"regexp"
	"strings"
	"sync"

	"code.google.com/p/go-uuid/uuid"

	"github.com/remind101/empire/repos"
)

var ErrInvalidName = errors.New("An app name must alphanumeric and dashes only, 3-30 chars in length.")

var NamePattern = regexp.MustCompile("^[a-z][a-z0-9-]{3,30}$")

// ID represents the unique identifier for an App.
type ID string

// Name represents the unique name for an App.
type Name string

// App represents an app.
type App struct {
	ID ID `json:"id"`

	Name Name `json:"name"`

	// The associated GitHub/Docker repo.
	Repo repos.Repo `json:"repo"`
}

func New(name Name, repo repos.Repo) (*App, error) {
	if !NamePattern.Match([]byte(name)) {
		return nil, ErrInvalidName
	}

	return &App{
		ID:   ID(uuid.NewRandom()),
		Name: name,
		Repo: repo,
	}, nil
}

// Repository represents a repository for creating and finding Apps.
type Repository interface {
	Create(*App) (*App, error)
	FindByID(ID) (*App, error)
	FindByRepo(repos.Repo) (*App, error)
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

func (r *repository) FindByID(id ID) (*App, error) {
	r.RLock()
	defer r.RUnlock()

	for _, app := range r.apps {
		if app.ID == id {
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

// Service provides methods for interacting with Apps.
type Service struct {
	Repository
}

// NewService returns a new Service instance.
func NewService(r Repository) *Service {
	if r == nil {
		r = newRepository()
	}

	return &Service{Repository: r}
}

func (s *Service) FindOrCreateByRepo(repo repos.Repo) (*App, error) {
	a, err := s.Repository.FindByRepo(repo)
	if err != nil {
		return a, err
	}

	if a == nil {
		a, err = New(nameFromRepo(repo), repo)
		if err != nil {
			return a, err
		}
		return s.Repository.Create(a)
	}

	return a, nil
}

func nameFromRepo(repo repos.Repo) Name {
	p := strings.Split(string(repo), "/")
	return Name(p[len(p)-1])
}
