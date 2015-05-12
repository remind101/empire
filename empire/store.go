package empire

import "github.com/jinzhu/gorm"

// store provides methods for CRUD'ing things.
type store struct {
	db *gorm.DB
}

func (s *store) Reset() error {
	var err error
	exec := func(sql string) {
		if err == nil {
			err = s.db.Exec(sql).Error
		}
	}

	exec(`TRUNCATE TABLE apps CASCADE`)
	exec(`TRUNCATE TABLE ports CASCADE`)
	exec(`INSERT INTO ports (port) (SELECT generate_series(9000,10000))`)

	return err
}

func (s *store) IsHealthy() bool {
	return s.db.DB().Ping() == nil
}
