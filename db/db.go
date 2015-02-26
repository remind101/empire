package db

import (
	"database/sql"
	"net/url"

	_ "github.com/lib/pq"
	gorp "gopkg.in/gorp.v1"
)

type DB struct {
	db    *sql.DB
	dbmap *gorp.DbMap
}

func NewDB(uri string) (*DB, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open(u.Scheme, uri)
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

	dbmap := &gorp.DbMap{Db: db, Dialect: dialect}
	//dbmap.TraceOn("[gorp]", log.New(os.Stdout, "myapp:", log.Lmicroseconds))

	return &DB{
		dbmap: dbmap,
		db:    db,
	}, nil
}

func (db *DB) AddTableWithName(v interface{}, name string) *gorp.TableMap {
	return db.dbmap.AddTableWithName(v, name)
}

func (db *DB) Insert(v ...interface{}) error {
	return db.dbmap.Insert(v...)
}

func (db *DB) Select(v interface{}, query string, args ...interface{}) error {
	_, err := db.dbmap.Select(v, query, args...)
	return err
}

func (db *DB) SelectOne(v interface{}, query string, args ...interface{}) error {
	return db.dbmap.SelectOne(v, query, args...)
}

func (db *DB) Exec(query string, args ...interface{}) (sql.Result, error) {
	return db.dbmap.Exec(query, args...)
}

func (db *DB) Begin() (*Transaction, error) {
	t, err := db.dbmap.Begin()
	if err != nil {
		return nil, err
	}

	return &Transaction{t}, nil
}

func (db *DB) Close() error {
	return db.db.Close()
}

type Transaction struct {
	*gorp.Transaction
}

func (t *Transaction) Select(v interface{}, query string, args ...interface{}) error {
	_, err := t.Transaction.Select(v, query, args...)
	return err
}
