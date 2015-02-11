package apps

import "github.com/remind101/empire/repos"

// ID represents the unique identifier for an App.
type ID string

// App represents an app.
type App struct {
	ID ID

	// The associated GitHub/Docker repo.
	Repo repos.Repo
}

// Repository represents a repository for creating and finding Apps.
type Repository interface {
	Create(*App) (*App, error)
	FindByID(ID) (*App, error)
	FindByRepo(repos.Repo) (*App, error)
}

// Service provides methods for interacting with Apps.
type Service struct {
	Repository
}

// NewService returns a new Service instance.
func NewService(r Repository) *Service {
	return &Service{}
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
