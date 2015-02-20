package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/repos"
)

// AppsService represents a service for interacting with Apps.
type AppsService interface {
	apps.Repository

	// FindOrCreateByRepo attempts to find an app by a repo name, or creates
	// a new app if it's not found.
	FindOrCreateByRepo(repos.Repo) (*apps.App, error)
}

// appsService is a base implementation of the AppsService interface.
type appsService struct {
	apps.Repository
}

// NewAppsService returns a new Service instance.
func NewAppsService(options Options) (AppsService, error) {
	return &appsService{
		Repository: apps.NewRepository(),
	}, nil
}

func (s *appsService) FindOrCreateByRepo(repo repos.Repo) (*apps.App, error) {
	a, err := s.Repository.FindByRepo(repo)
	if err != nil {
		return a, err
	}

	// If the app wasn't found, create a new up linked to this repo.
	if a == nil {
		a, err := apps.NewFromRepo(repo)
		if err != nil {
			return a, err
		}
		return s.Repository.Create(a)
	}

	return a, nil
}
