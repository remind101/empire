package empire

import (
	"database/sql"
	"fmt"
	"net/url"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire/pkg/headerutil"
	"github.com/remind101/migrate"
)

// Empire only supports postgres at the moment.
const DBDriver = "postgres"

// IncompatibleSchemaError is an error that gets returned from
// CheckSchemaVersion.
type IncompatibleSchemaError struct {
	SchemaVersion         int
	ExpectedSchemaVersion int
}

// Error implements the error interface.
func (e *IncompatibleSchemaError) Error() string {
	return fmt.Sprintf("expected database schema to be at version %d, but was %d", e.ExpectedSchemaVersion, e.SchemaVersion)
}

// DB wraps a gorm.DB and provides the datastore layer for Empire.
type DB struct {
	*gorm.DB

	uri string

	migrator *migrate.Migrator
}

// OpenDB returns a new gorm.DB instance.
func OpenDB(uri string) (*DB, error) {
	_, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open(DBDriver, uri)
	if err != nil {
		return nil, err
	}

	return NewDB(conn)
}

// NewDB wraps a sql.DB instance as a DB.
func NewDB(conn *sql.DB) (*DB, error) {
	db, err := gorm.Open(DBDriver, conn)
	if err != nil {
		return nil, err
	}

	m := migrate.NewPostgresMigrator(conn)
	// Run all migrations in a single transaction, so they will be rolled
	// back as one. This is almost always the behavior that users would want
	// when upgrading Empire. If a new release has multiple migrations, and
	// one of those fails, it's easier for them if the entire upgrade rolls
	// back instead of getting stuck in failed state.
	m.TransactionMode = migrate.SingleTransaction

	return &DB{
		DB:       &db,
		migrator: m,
	}, nil
}

// MigrateUp migrates the database to the latest version of the schema.
func (db *DB) MigrateUp() error {
	return db.migrator.Exec(migrate.Up, migrations...)
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
	exec(`TRUNCATE TABLE slugs CASCADE`)
	exec(`INSERT INTO ports (port) (SELECT generate_series(9000,10000))`)

	return err
}

// IsHealthy checks that we can connect to the database.
func (db *DB) IsHealthy() error {
	if err := db.DB.DB().Ping(); err != nil {
		return err
	}

	if err := db.CheckSchemaVersion(); err != nil {
		return err
	}

	return nil
}

// CheckSchemaVersion verifies that the actual database schema matches the
// version that this version of Empire expects.
func (db *DB) CheckSchemaVersion() error {
	schemaVersion, err := db.SchemaVersion()
	if err != nil {
		return fmt.Errorf("error fetching schema version: %v", err)
	}

	expectedSchemaVersion := latestSchema()
	if schemaVersion != expectedSchemaVersion {
		return &IncompatibleSchemaError{
			SchemaVersion:         schemaVersion,
			ExpectedSchemaVersion: expectedSchemaVersion,
		}
	}

	return nil
}

// SchemaVersion returns the current schema version.
func (db *DB) SchemaVersion() (int, error) {
	sql := `select version from schema_migrations order by version desc limit 1`
	var schemaVersion int
	err := db.DB.DB().QueryRow(sql).Scan(&schemaVersion)
	return schemaVersion, err
}

// Debug puts the db in debug mode, which logs all queries.
func (db *DB) Debug() {
	db.DB = db.DB.Debug()
}

// scope is an interface that scopes a gorm.DB. Scopes are used in
// ThingsFirst and ThingsAll methods on the store for filtering/querying.
type scope interface {
	scope(*gorm.DB) *gorm.DB
}

// scopeFunc implements the scope interface for functions.
type scopeFunc func(*gorm.DB) *gorm.DB

// scope implements the scope interface.
func (f scopeFunc) scope(db *gorm.DB) *gorm.DB {
	return f(db)
}

// idEquals returns a scope that will find the item by id.
func idEquals(id string) scope {
	return fieldEquals("id", id)
}

// forApp returns a scope that will filter items belonging the the given app.
func forApp(app *App) scope {
	return fieldEquals("app_id", app.ID)
}

// composedScope is an implementation of the Scope interface that chains the
// scopes together.
type composedScope []scope

// scope implements the scope interface.
func (s composedScope) scope(db *gorm.DB) *gorm.DB {
	for _, s := range s {
		db = s.scope(db)
	}

	return db
}

// fieldEquals returns a scope that filters on a field.
func fieldEquals(field string, v interface{}) scope {
	return scopeFunc(func(db *gorm.DB) *gorm.DB {
		return db.Where(fmt.Sprintf("%s = ?", field), v)
	})
}

// preload returns a scope that preloads the associations.
func preload(associations ...string) scope {
	var scope composedScope

	for _, a := range associations {
		aa := a
		scope = append(scope, scopeFunc(func(db *gorm.DB) *gorm.DB {
			return db.Preload(aa)
		}))
	}

	return scope
}

// order returns a scope that orders the results.
func order(order string) scope {
	return scopeFunc(func(db *gorm.DB) *gorm.DB {
		return db.Order(order)
	})
}

// limit returns a scope that limits the results.
func limit(limit int) scope {
	return scopeFunc(func(db *gorm.DB) *gorm.DB {
		return db.Limit(limit)
	})
}

// inRange returns a scope that limits and orders the results.
func inRange(r headerutil.Range) scope {
	var scope composedScope

	if r.Max != nil {
		scope = append(scope, limit(*r.Max))
	}

	if r.Sort != nil && r.Order != nil {
		o := fmt.Sprintf("%s %s", *r.Sort, *r.Order)
		scope = append(scope, order(o))
	}

	return scope
}

// first is a small helper that finds the first record matching a scope, and
// returns the error.
func first(db *gorm.DB, scope scope, v interface{}) error {
	return scope.scope(db).First(v).Error
}

// find is a small helper that finds records matching the scope, and returns the
// error.
func find(db *gorm.DB, scope scope, v interface{}) error {
	return scope.scope(db).Find(v).Error
}
