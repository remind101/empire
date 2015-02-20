package empire

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/processes"
)

type FormationsService interface {
	formations.Repository
	Scale(*apps.App, processes.Type, int) (*formations.Formation, error)
}

// formationsService is a service for managing process formations for apps.
type formationsService struct {
	formations.Repository
}

// NewFormationsService returns a new Service instance.
func NewFormationsService(r formations.Repository) FormationsService {
	if r == nil {
		r = formations.NewRepository()
	}

	return &formationsService{
		Repository: r,
	}
}

// Scale a given process type up or down.
func (s *formationsService) Scale(app *apps.App, pt processes.Type, count int) (*formations.Formation, error) {
	fmtns, err := s.Repository.Get(app)
	if err != nil {
		return nil, err
	}

	if fmtns == nil {
		fmtns = make(formations.Formations)
	}

	f := findFormation(fmtns, pt)
	f.Count = count

	if err := s.Repository.Set(app, fmtns); err != nil {
		return f, err
	}

	return f, nil
}

// findFormation finds a Formation for a processes.Type, or builds a new one if
// it's not found.
func findFormation(fmtns formations.Formations, pt processes.Type) *formations.Formation {
	if f, found := fmtns[pt]; found {
		return f
	}

	f := formations.NewFormation(pt)
	fmtns[pt] = f
	return f
}
