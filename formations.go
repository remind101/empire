package empire

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

func NewFormationsRepository(db DB) (FormationsRepository, error) {
	return &formationsRepository{db}, nil
}

// dbFormation is the database representation of a Formation.
type dbFormation struct {
	ID string `db:"id"`
}

// formationsRepository is an implementation of the FormationsRepository interface backed by
// a DB.
type formationsRepository struct {
	DB
}

func (r *formationsRepository) Find(id FormationID) (*Formation, error) {
	var f dbFormation

	if err := r.DB.SelectOne(&f, `select * from formations where id = $1`, string(id)); err != nil {
		return nil, err
	}

	return toFormation(&f, nil), nil
}

func (r *formationsRepository) Create(formation *Formation) (*Formation, error) {
	f := fromFormation(formation)

	if err := r.DB.Insert(f); err != nil {
		return formation, err
	}

	return toFormation(f, formation), nil
}

func fromFormation(formation *Formation) *dbFormation {
	return &dbFormation{
		ID: string(formation.ID),
	}
}

func toFormation(f *dbFormation, formation *Formation) *Formation {
	if formation == nil {
		formation = &Formation{}
	}

	formation.ID = FormationID(f.ID)

	return formation
}

// FormationsService represents an interface for interacting with Formations.
type FormationsService interface {
	FormationsRepository
}

// NewFormationsService returns a new FormationsService instance.
func NewFormationsService(f FormationsRepository, p ProcessesRepository) (FormationsService, error) {
	return &formationsService{
		FormationsRepository: f,
		ProcessesRepository:  p,
	}, nil
}

// formationsService is a base implementation of the FormationsService.
type formationsService struct {
	FormationsRepository
	ProcessesRepository
}

func (s *formationsService) Find(id FormationID) (*Formation, error) {
	f, err := s.FormationsRepository.Find(id)
	if err != nil {
		return f, err
	}

	p, err := s.ProcessesRepository.All(id)
	if err != nil {
		return f, err
	}

	f.Processes = p

	return f, nil
}

// Create first creates the Formation, then Creates all of the Processes.
func (s *formationsService) Create(formation *Formation) (*Formation, error) {
	f, err := s.FormationsRepository.Create(formation)
	if err != nil {
		return f, err
	}

	for t, p := range formation.Processes {
		p.Formation = f

		if _, _, err := s.ProcessesRepository.Create(t, p); err != nil {
			return f, err
		}
	}

	return f, nil
}
