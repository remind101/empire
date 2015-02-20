package formations

import (
	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/processes"
)

// Formations maps a ProcessType to a Formation definition.
type Formations map[processes.Type]*Formation

// Formation represents configuration for a process type.
type Formation struct {
	ProcessType processes.Type

	Count int // Count represents the desired number of processes to run.

	// Size Size // The size of the instance to put these processes on.
}

// CommandFormation is a composition of a Formation and a processes.Command.
type CommandFormation struct {
	*Formation
	Command processes.Command
}

// NewFormation returns a new Formation with an appropriate default Count.
func NewFormation(pt processes.Type) *Formation {
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

func NewRepository() Repository {
	return newRepository()
}

// repository is an in memory implementation of the Repository interface.
type repository struct {
	formations map[apps.Name]Formations
}

// newRepository returns a new repository instance.
func newRepository() *repository {
	return &repository{
		formations: make(map[apps.Name]Formations),
	}
}

// Set sets the process formations for the given app.
func (r *repository) Set(app *apps.App, f Formations) error {
	r.formations[app.Name] = f
	return nil
}

// Get gets the current process formations for the given app.
func (r *repository) Get(app *apps.App) (Formations, error) {
	return r.formations[app.Name], nil
}
