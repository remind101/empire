package lb

import (
	"database/sql"
	"testing"

	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

func TestDBPortAllocator_Get(t *testing.T) {
	db := newDB(t)
	a := &DBPortAllocator{
		db: db,
	}

	port, err := a.Get()
	assert.NoError(t, err)
	assert.NotEqual(t, 0, port)

	err = a.Put(port)
	assert.NoError(t, err)
}

func newDB(t testing.TB) *sql.DB {
	db, err := sql.Open("postgres", "postgres://localhost/empire?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}
	return db
}
