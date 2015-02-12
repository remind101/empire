package apps

import (
	"strconv"
	"sync"

	"github.com/remind101/empire/repos"
)

// ID represents the unique identifier for an App.
type ID string

// App represents an app.
type App struct {
	ID ID `json:"id"`

	// The associated GitHub/Docker repo.
	Repo repos.Repo `json:"repo"`
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

	r.id++
	app.ID = ID(strconv.Itoa(r.id))
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
		return s.Repository.Create(&App{
			Repo: repo,
		})
	}

	return a, nil
}
