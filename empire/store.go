package empire

// store provides methods for CRUD'ing things.
type store struct {
	db *db
}

func (s *store) Reset() error {
	_, err := s.db.Exec(`TRUNCATE TABLE apps CASCADE`)
	return err
}

func (s *store) IsHealthy() bool {
	if _, err := s.db.Exec(`SELECT 1`); err != nil {
		return false
	}

	return true
}
