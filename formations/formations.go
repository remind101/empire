package formations

import (
	"strconv"

	"github.com/remind101/empire/processes"
)

// ID represents a unique identifier for a Formation.
type ID string

// Formation represents a collection of configured Processes.
type Formation struct {
	ID ID `json:"id"`

	// Configured processes in this formation.
	Processes processes.ProcessMap `json:"processes"`
}

// Repository is an interface for creating and finding Formations.
type Repository interface {
	// Find finds a Formation by it's ID.
	Find(ID) (*Formation, error)

	// Create creates a new Formation.
	Create(*Formation) (*Formation, error)
}

func NewRepository() Repository {
	return newRepository()
}

type repository struct {
	formations map[ID]*Formation
	id         int
}

func newRepository() *repository {
	return &repository{
		formations: make(map[ID]*Formation),
	}
}

func (r *repository) Find(id ID) (*Formation, error) {
	return r.formations[id], nil
}

func (r *repository) Create(formation *Formation) (*Formation, error) {
	r.id++

	formation.ID = ID(strconv.Itoa(r.id))

	return formation, nil
}
