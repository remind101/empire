package empire

import "github.com/remind101/empire/formations"

type mockFormationsRepository struct {
	FindFunc   func(formations.ID) (*formations.Formation, error)
	CreateFunc func(*formations.Formation) (*formations.Formation, error)
}

func (r *mockFormationsRepository) Find(id formations.ID) (*formations.Formation, error) {
	if r.FindFunc != nil {
		return r.FindFunc(id)
	}

	return nil, nil
}

func (r *mockFormationsRepository) Create(formation *formations.Formation) (*formations.Formation, error) {
	if r.CreateFunc != nil {
		return r.CreateFunc(formation)
	}

	return formation, nil
}

type mockFormationsService struct {
	mockFormationsRepository
}
