package tugboat

type store struct {
	db *db
}

func (s *store) Reset() error {
	_, err := s.db.Exec(`TRUNCATE TABLE deployments CASCADE`)
	return err
}
