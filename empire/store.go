package empire

// store provides methods for CRUD'ing things.
type store struct {
	db *db
}

func (s *store) Reset() error {
	var err error
	exec := func(sql string) {
		if err == nil {
			_, err = s.db.Exec(sql)
		}
	}

	exec(`TRUNCATE TABLE apps CASCADE`)
	exec(`TRUNCATE TABLE ports CASCADE`)
	exec(`INSERT INTO ports (port) (SELECT generate_series(9000,10000))`)

	return err
}

func (s *store) IsHealthy() bool {
	if _, err := s.db.Exec(`SELECT 1`); err != nil {
		return false
	}

	return true
}
