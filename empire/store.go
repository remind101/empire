package empire

// store provides methods for CRUD'ing things.
type store struct {
	db *db
}

func (s *store) Reset() error {
	_, err := s.db.Exec(`TRUNCATE TABLE apps CASCADE`)
	return err
}
