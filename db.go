package empire

import (
	"database/sql"
	"net/url"

	"github.com/jinzhu/gorm"
)

// NewDB returns a new gorm.DB instance.
func NewDB(uri string) (*gorm.DB, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open(u.Scheme, uri)
	if err != nil {
		return nil, err
	}

	db, err := gorm.Open(u.Scheme, conn)
	if err != nil {
		return nil, err
	}

	return &db, nil
}
