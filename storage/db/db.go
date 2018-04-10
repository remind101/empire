package db

import (
	"fmt"

	"github.com/jinzhu/gorm"
	"github.com/remind101/empire"
	"github.com/remind101/empire/pkg/headerutil"
)

// Storage implements the empire.Storage interface backed by a SQL database
// (Postgres only right now).
type Storage struct {
	DB *empire.DB
	db *gorm.DB
}

func New(db *empire.DB) *Storage {
	return &Storage{
		DB: db,
		db: db.DB,
	}
}

type AppsQuery empire.AppsQuery

// scope implements the scope interface.
func (q AppsQuery) scope(db *gorm.DB) *gorm.DB {
	var scope composedScope

	if q.ID != nil {
		scope = append(scope, idEquals(*q.ID))
	}

	if q.Name != nil {
		scope = append(scope, fieldEquals("name", *q.Name))
	}

	if q.Repo != nil {
		scope = append(scope, fieldEquals("repo", *q.Repo))
	}

	return scope.scope(db)
}

func (s *Storage) AppsFind(q empire.AppsQuery) (*empire.App, error) {
	return appsFind(s.db, AppsQuery(q))
}

func (s *Storage) Apps(q empire.AppsQuery) ([]*empire.App, error) {
	return apps(s.db, AppsQuery(q))
}

// appsFind finds a single app given the scope.
func appsFind(db *gorm.DB, scope scope) (*empire.App, error) {
	var app empire.App
	return &app, first(db, scope, &app)
}

// apps finds all apps matching the scope.
func apps(db *gorm.DB, scope scope) ([]*empire.App, error) {
	var apps []*empire.App
	// Default to ordering by name.
	scope = composedScope{order("name"), scope}
	return apps, find(db, scope, &apps)
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
func forApp(app *empire.App) scope {
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
