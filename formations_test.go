package empire

type mockFormationsRepository struct {
	FindFunc   func(FormationID) (*Formation, error)
	CreateFunc func(*Formation) (*Formation, error)
}

func (r *mockFormationsRepository) Find(id FormationID) (*Formation, error) {
	if r.FindFunc != nil {
		return r.FindFunc(id)
	}

	return nil, nil
}

func (r *mockFormationsRepository) Create(formation *Formation) (*Formation, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(formation)
	}

	return formation, nil
}
