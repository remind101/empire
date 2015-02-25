package empire

import (
	"code.google.com/p/go-uuid/uuid"
	"github.com/remind101/empire/stores"
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

// NewFormationsRepository returns a FormationsRepository backed by an in memory store
func NewFormationsRepository() FormationsRepository {
	return &formationsRepository{stores.NewMemStore()}
}

// NewEtcdFormationsRepository returns a FormationsRepository backed by etcd
func NewEtcdFormationsRepository(ns string) (FormationsRepository, error) {
	s, err := stores.NewEtcdStore(ns)
	if err != nil {
		return nil, err
	}
	return &formationsRepository{s}, nil
}

type formationsRepository struct {
	s stores.Store
}

func (r *formationsRepository) Find(id FormationID) (*Formation, error) {
	f := &Formation{}

	if ok, err := r.s.Get(string(id), f); err != nil || !ok {
		return nil, err
	}

	return f, nil
}

func (r *formationsRepository) Create(formation *Formation) (*Formation, error) {
	// TODO make formation.ID `App.ID + Release.Version`
	if formation.ID == "" {
		formation.ID = FormationID(uuid.NewRandom())
	}

	if err := r.s.Set(string(formation.ID), formation); err != nil {
		return formation, err
	}

	return formation, nil
}
