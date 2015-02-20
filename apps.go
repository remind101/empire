package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/repos"
)

type AppsService interface {
	apps.Repository

	FindOrCreateByRepo(repos.Repo) (*apps.App, error)
}

// appsService provides methods for interacting with Apps.
type appsService struct {
	apps.Repository
}

// NewAppsService returns a new Service instance.
func NewAppsService(r apps.Repository) AppsService {
	if r == nil {
		r = apps.NewRepository()
	}

	return &appsService{
		Repository: r,
	}
}

func (s *appsService) FindOrCreateByRepo(repo repos.Repo) (*apps.App, error) {
	a, err := s.Repository.FindByRepo(repo)
	if err != nil {
		return a, err
	}

	if a == nil {
		a, err = apps.New(apps.NewNameFromRepo(repo), repo)
		if err != nil {
			return a, err
		}
		return s.Repository.Create(a)
	}

	return a, nil
}
