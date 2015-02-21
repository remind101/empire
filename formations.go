package empire

import (
	"errors"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/processes"
	"github.com/remind101/empire/slugs"
)

var (
	// Scaling errors
	ErrNoFormation        = errors.New("no process formation for app")
	ErrInvalidProcessType = errors.New("no matching process type")
)

// FormationsService represents a service for configuring the apps process
// formation.
type FormationsService interface {
	formations.Repository

	// GetOrCreate gets an apps formation, or creates an initial formation
	// based on a slugs available process types.
	GetOrCreate(*apps.App, *slugs.Slug) (formations.Formations, error)

	// Scale scales a process type for an app.
	Scale(*apps.App, processes.Type, int) (*formations.Formation, error)
}

// formationsService is a service for managing process formations for apps.
type formationsService struct {
	formations.Repository
}

// NewFormationsService returns a new Service instance.
func NewFormationsService(options Options) (FormationsService, error) {
	return &formationsService{
		Repository: formations.NewRepository(),
	}, nil
}

func (s *formationsService) GetOrCreate(app *apps.App, slug *slugs.Slug) (formations.Formations, error) {
	f, err := s.Get(app)
	if err != nil {
		if err != ErrNoFormation {
			// Something really went wrong, return it.
			return f, err
		}

		// So the app doesn't have a formation yet, which means it's
		// new. Let's create an initial formation for it.
		f := newFormation(slug)
		return f, s.Set(app, f)
	}

	return f, nil
}

// Scale scales a given process type up or down.
func (s *formationsService) Scale(app *apps.App, pt processes.Type, count int) (*formations.Formation, error) {
	fmtns, err := s.Repository.Get(app)
	if err != nil {
		return nil, err
	}

	// If the app doesn't have a process formation yet, it means that a
	// release has never been created. We don't want to allow scaling in
	// these cases.
	if fmtns == nil {
		return nil, ErrNoFormation
	}

	// If the provided process type is not in this apps formation, then we
	// shouldn't be able to scale it.
	f, found := fmtns[pt]
	if !found {
		return nil, ErrInvalidProcessType
	}

	// Set the instance count for this formation.
	f.Count = count

	if err := s.Repository.Set(app, fmtns); err != nil {
		return f, err
	}

	return f, nil
}

func newFormation(slug *slugs.Slug) formations.Formations {
	f := make(formations.Formations)

	for pt, _ := range slug.ProcessTypes {
		f[pt] = formations.NewFormation(pt)
	}

	return f
}
