package empire

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/remind101/migrate"
	"github.com/stretchr/testify/assert"
)

// Tests migrating the database down, then back up again.
func TestMigrations(t *testing.T) {
	db, err := OpenDB("postgres://localhost/empire?sslmode=disable")
	if err != nil {
		t.Fatal(err)
	}

	err = db.migrator.Exec(migrate.Up, Migrations...)
	assert.NoError(t, err)

	err = db.Reset()
	assert.NoError(t, err)

	err = db.migrator.Exec(migrate.Down, Migrations...)
	assert.NoError(t, err)

	err = db.migrator.Exec(migrate.Up, Migrations...)
	assert.NoError(t, err)
}

func TestLatestSchema(t *testing.T) {
	assert.Equal(t, 17, latestSchema())
}
