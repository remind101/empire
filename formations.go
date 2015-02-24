package empire

import (
	"strconv"
)

// FormationID represents a unique identifier for a Formation.
type FormationID string

// Formation represents a collection of configured Processes.
type Formation struct {
	ID FormationID `json:"id"`

	// Configured processes in this formation.
	Processes ProcessMap `json:"processes"`
}

// FormationsRepository is an interface for creating and finding Formations.
type FormationsRepository interface {
	// Find finds a Formation by it's ID.
	Find(FormationID) (*Formation, error)

	// Create creates a new Formation.
	Create(*Formation) (*Formation, error)
}

func NewFormationsRepository() FormationsRepository {
	return newFormationsRepository()
}

type formationsRepository struct {
	formations map[FormationID]*Formation
	id         int
}

func newFormationsRepository() *formationsRepository {
	return &formationsRepository{
		formations: make(map[FormationID]*Formation),
	}
}

func (r *formationsRepository) Find(id FormationID) (*Formation, error) {
	return r.formations[id], nil
}

func (r *formationsRepository) Create(formation *Formation) (*Formation, error) {
	r.id++

	formation.ID = FormationID(strconv.Itoa(r.id))

	return formation, nil
}
