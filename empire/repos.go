package empire

import (
	"database/sql/driver"
	"strings"
)

type Repo string

// Scan implements the sql.Scanner interface.
func (r *Repo) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*r = Repo(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (r *Repo) Value() (driver.Value, error) {
	if r == nil {
		return driver.Value(nil), nil
	}

	return driver.Value(string(*r)), nil
}

func (r Repo) Domain() string {
	domain, _ := r.Split()
	return domain
}

func (r Repo) Path() string {
	_, path := r.Split()
	return path
}

// Split splits the repo into the Domain and Path segments.
func (r Repo) Split() (string, string) {
	parts := strings.Split(string(r), "/")

	if len(parts) == 2 {
		return "", strings.Join(parts, "/")
	}

	return parts[0], strings.Join(parts[1:], "/")
}
