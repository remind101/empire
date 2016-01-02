package tugboat

import (
	"database/sql"
	"net/url"

	_ "github.com/lib/pq"
	"gopkg.in/gorp.v1"
)

type db struct {
	*gorp.DbMap
	db *sql.DB
}

func dialDB(uri string) (*db, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	conn, err := sql.Open(u.Scheme, uri)
	if err != nil {
		return nil, err
	}

	dbmap := &gorp.DbMap{Db: conn, Dialect: gorp.PostgresDialect{}}
	db := &db{DbMap: dbmap, db: conn}

	db.DbMap.AddTableWithName(Deployment{}, "deployments").SetKeys(true, "ID")
	db.DbMap.AddTableWithName(LogLine{}, "logs").SetKeys(true, "ID")

	return db, nil
}
