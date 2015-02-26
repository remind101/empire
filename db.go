package empire

import (
	"database/sql"

	"github.com/remind101/empire/db"
)

type Inserter interface {
	// Insert inserts a record.
	Insert(...interface{}) error
}

type Execer interface {
	// Exec executes an arbitrary SQL query.
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Queryier interface {
	// Select performs a query and populates the interface with the
	// returned records. interface must be a pointer to a slice
	Select(interface{}, string, ...interface{}) error

	// SelectOne performs a query and populates the interface with the
	// returned record.
	SelectOne(interface{}, string, ...interface{}) error
}

// DB represents an interface for performing queries against a SQL db.
type DB interface {
	Inserter
	Execer
	Queryier

	// Begin opens a transaction.
	Begin() (*db.Transaction, error)

	// Close closes the db.
	Close() error
}

// NewDB returns a new DB instance with table mappings configured.
func NewDB(uri string) (DB, error) {
	db, err := db.NewDB(uri)
	if err != nil {
		return db, err
	}

	db.AddTableWithName(App{}, "apps")
	db.AddTableWithName(dbConfig{}, "configs").SetKeys(true, "ID")
	db.AddTableWithName(dbSlug{}, "slugs").SetKeys(true, "ID")
	db.AddTableWithName(dbProcess{}, "processes").SetKeys(true, "ID")
	db.AddTableWithName(dbRelease{}, "releases").SetKeys(true, "ID")
	db.AddTableWithName(dbJob{}, "jobs").SetKeys(true, "ID")

	return db, nil
}
