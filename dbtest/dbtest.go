package dbtest

import (
	"database/sql"
	"flag"
	"os"
	"testing"
)

var DatabaseURL = flag.String("db.url", getenv("TEST_DATABASE_URL", "postgres://localhost/empire?sslmode=disable"), "A connection string where a postgres instance is running.")

func Open(t testing.TB) *sql.DB {
	db, err := sql.Open("postgres", *DatabaseURL)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func getenv(key, fallback string) string {
	v, ok := os.LookupEnv(key)
	if ok {
		return v
	}
	return fallback
}
