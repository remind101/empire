package empire

import (
	"database/sql"
	"fmt"
	"net/url"

	gorp "gopkg.in/gorp.v1"
)

type SqlExecutor gorp.SqlExecutor

type db struct {
	db    *sql.DB
	dbmap *gorp.DbMap
}

// newDB returns a new db instance with table mappings configured.
func newDB(uri string) (*db, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open(u.Scheme, uri)
	if err != nil {
		return nil, err
	}

	var dialect gorp.Dialect
	switch u.Scheme {
	case "postgres":
		dialect = gorp.PostgresDialect{}
	default:
		dialect = gorp.SqliteDialect{}
	}

	dbmap := &gorp.DbMap{Db: conn, Dialect: dialect}
	db := &db{dbmap: dbmap, db: conn}

	db.AddTableWithName(App{}, "apps").SetKeys(false, "Name").SetKeys(true, "ID")
	db.AddTableWithName(Certificate{}, "certificates").SetKeys(true, "ID")
	db.AddTableWithName(Config{}, "configs").SetKeys(true, "ID")
	db.AddTableWithName(Domain{}, "domains").SetKeys(true, "ID")
	db.AddTableWithName(Port{}, "ports").SetKeys(true, "ID")
	db.AddTableWithName(Process{}, "processes").SetKeys(true, "ID")
	db.AddTableWithName(Release{}, "releases").SetKeys(true, "ID")
	db.AddTableWithName(Slug{}, "slugs").SetKeys(true, "ID")

	return db, nil
}

func (db *db) AddTableWithName(v interface{}, name string) *gorp.TableMap {
	return db.dbmap.AddTableWithName(v, name)
}

func (db *db) Insert(v ...interface{}) error {
	return db.dbmap.Insert(v...)
}

func (db *db) Update(v ...interface{}) (int64, error) {
	return db.dbmap.Update(v...)
}

func (db *db) Delete(list ...interface{}) (int64, error) {
	return db.dbmap.Delete(list...)
}

func (db *db) Select(v interface{}, query string, args ...interface{}) error {
	_, err := db.dbmap.Select(v, query, args...)
	return err
}

func (db *db) SelectOne(v interface{}, query string, args ...interface{}) error {
	return db.dbmap.SelectOne(v, query, args...)
}

func (db *db) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.dbmap.Exec(query, args...)
}

func (db *db) Begin() (*Transaction, error) {
	t, err := db.dbmap.Begin()
	if err != nil {
		return nil, err
	}

	return &Transaction{t}, nil
}

func (db *db) Close() error {
	return db.db.Close()
}

type Transaction struct {
	*gorp.Transaction
}

func (t *Transaction) Select(v interface{}, query string, args ...interface{}) error {
	_, err := t.Transaction.Select(v, query, args...)
	return err
}

func findBy(db *db, v interface{}, table, field string, value interface{}) error {
	q := fmt.Sprintf(`select * from %s where %s = $1 limit 1`, table, field)

	return db.SelectOne(v, q, value)
}
