package empire

import "database/sql/driver"

type Repo string

// Scan implements the sql.Scanner interface.
func (r *Repo) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*r = Repo(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (r Repo) Value() (driver.Value, error) {
	return driver.Value(string(r)), nil
}
