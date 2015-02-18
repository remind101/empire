package formations

import "github.com/remind101/empire/apps"

// ProcessType represents the type of a given process/command.
type ProcessType string

// Formations maps a ProcessType to a Formation definition.
type Formations map[ProcessType]*Formation

// Formation represents configuration for a process type.
type Formation struct {
	ProcessType ProcessType

	Count int // Count represents the desired number of processes to run.

	// Size Size // The size of the instance to put these processes on.
}

// NewFormation returns a new Formation with an appropriate default Count.
func NewFormation(pt ProcessType) *Formation {
	count := 0

	if pt == "web" {
		count = 1
	}

	return &Formation{
		ProcessType: pt,
		Count:       count,
	}
}

// Repository is an interface that can store and retrieve formations for apps.
type Repository interface {
	// Set sets the apps desired process formations.
	Set(*apps.App, Formations) error

	// Get gets the apps desired process formations.
	Get(*apps.App) (Formations, error)
}

// repository is an in memory implementation of the Repository interface.
type repository struct {
	formations map[apps.ID]Formations
}

// newRepository returns a new repository instance.
func newRepository() *repository {
	return &repository{
		formations: make(map[apps.ID]Formations),
	}
}

// Set sets the process formations for the given app.
func (r *repository) Set(app *apps.App, f Formations) error {
	r.formations[app.ID] = f
	return nil
}

// Get gets the current process formations for the given app.
func (r *repository) Get(app *apps.App) (Formations, error) {
	if _, found := r.formations[app.ID]; !found {
		r.formations[app.ID] = make(Formations)
	}

	return r.formations[app.ID], nil
}

// Service is a service for managing process formations for apps.
type Service struct {
	Repository
}

// Scale a given process type up or down.
func (s *Service) Scale(app *apps.App, pt ProcessType, count int) (*Formation, error) {
	formations, err := s.Repository.Get(app)
	if err != nil {
		return nil, err
	}

	f := findFormation(formations, pt)
	f.Count = count

	if err := s.Repository.Set(app, formations); err != nil {
		return f, err
	}

	return f, nil
}

// findFormation finds a Formation for a ProcessType, or builds a new one if
// it's not found.
func findFormation(formations Formations, pt ProcessType) *Formation {
	if f, found := formations[pt]; found {
		return f
	}

	f := NewFormation(pt)
	formations[pt] = f
	return f
}
