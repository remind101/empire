package empire

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/migrate"
)

// DB wraps a gorm.DB and provides the datastore layer for Empire.
type DB struct {
	*gorm.DB

	uri string
}

// OpenDB returns a new gorm.DB instance.
func OpenDB(uri string) (*DB, error) {
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

	return &DB{
		DB:  &db,
		uri: uri,
	}, nil
}

// MigrateUp migrates the database to the latest version of the schema.
func (db *DB) MigrateUp() error {
	return migrate.Exec(db.DB.DB(), migrate.Up, Migrations...)
}

// Reset resets the database to a pristine state.
func (db *DB) Reset() error {
	var err error
	exec := func(sql string) {
		if err == nil {
			err = db.Exec(sql).Error
		}
	}

	exec(`TRUNCATE TABLE apps CASCADE`)
	exec(`TRUNCATE TABLE ports CASCADE`)
	exec(`INSERT INTO ports (port) (SELECT generate_series(9000,10000))`)

	return err
}

// IsHealthy checks that we can connect to the database.
func (db *DB) IsHealthy() bool {
	return db.DB.DB().Ping() == nil
}

// Debug puts the db in debug mode, which logs all queries.
func (db *DB) Debug() {
	db.DB = db.DB.Debug()
}

// Scope is an interface that scopes a gorm.DB. Scopes are used in
// ThingsFirst and ThingsAll methods on the store for filtering/querying.
type Scope interface {
	Scope(*gorm.DB) *gorm.DB
}

// ScopeFunc implements the Scope interface for functions.
type ScopeFunc func(*gorm.DB) *gorm.DB

// Scope implements the Scope interface.
func (f ScopeFunc) Scope(db *gorm.DB) *gorm.DB {
	return f(db)
}

// All returns a scope that simply returns the db.
var All = ScopeFunc(func(db *gorm.DB) *gorm.DB {
	return db
})

// ID returns a Scope that will find the item by id.
func ID(id string) Scope {
	return FieldEquals("id", id)
}

// ForApp returns a Scope that will filter items belonging the the given app.
func ForApp(app *App) Scope {
	return FieldEquals("app_id", app.ID)
}

// ComposedScope is an implementation of the Scope interface that chains the
// scopes together.
type ComposedScope []Scope

// Scope implements the Scope interface.
func (s ComposedScope) Scope(db *gorm.DB) *gorm.DB {
	for _, s := range s {
		db = s.Scope(db)
	}

	return db
}

// FieldEquals returns a Scope that filters on a field.
func FieldEquals(field string, v interface{}) Scope {
	return ScopeFunc(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s = ?", field), v)
	})
}

// Preload returns a Scope that preloads the associations.
func Preload(associations ...string) Scope {
	var scope ComposedScope

	for _, a := range associations {
		aa := a
		scope = append(scope, ScopeFunc(func(db *gorm.DB) *gorm.DB {
			return db.Preload(aa)
		}))
	}

	return scope
}

// Order returns a Scope that orders the results.
func Order(order string) Scope {
	return ScopeFunc(func(db *gorm.DB) *gorm.DB {
		return db.Order(order)
	})
}

// Limit returns a Scope that limits the results.
func Limit(limit int) Scope {
	return ScopeFunc(func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	})
}

// Range returns a Scope that limits and orders the results.
func Range(r headerutil.Range) Scope {
	var scope ComposedScope

	if r.Max != nil {
		scope = append(scope, Limit(*r.Max))
	}

	if r.Sort != nil && r.Order != nil {
		order := fmt.Sprintf("%s %s", *r.Sort, *r.Order)
		scope = append(scope, Order(order))
	}

	return scope
}

// first is a small helper that finds the first record matching a scope, and
// returns the error.
func first(db *gorm.DB, scope Scope, v interface{}) error {
	return scope.Scope(db).First(v).Error
}

// find is a small helper that finds records matching the scope, and returns the
// error.
func find(db *gorm.DB, scope Scope, v interface{}) error {
	return scope.Scope(db).Find(v).Error
}
