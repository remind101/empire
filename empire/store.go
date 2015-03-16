package empire

// Store provides methods for CRUD'ing things.
type Store struct {
	db *db
}

func (s *Store) Reset() error {
	_, err := s.db.Exec(`TRUNCATE TABLE apps CASCADE`)
	return err
}
