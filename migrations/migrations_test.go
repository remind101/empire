package migrations_test

import (
	"testing"

	"github.com/remind101/empire/empiretest"
	"github.com/stretchr/testify/assert"
)

// Run the tests with empiretest.Run, which will lock access to the database
// since it can't be shared by parallel tests.
func TestMain(m *testing.M) {
	empiretest.Run(m)
}

func TestMigration(t *testing.T) {
	db := empiretest.OpenDB(t)
	err := db.MigrateDown()
	assert.NoError(t, err)

	err = db.MigrateUp()
	assert.NoError(t, err)

	err = db.MigrateDown()
	assert.NoError(t, err)

	// Now check that we have no tables other than the `gorp_migrations`
	// table.
	rows, err := db.DB.DB().Query(`SELECT table_name FROM information_schema.tables WHERE table_type = 'BASE TABLE' AND table_schema = 'public' ORDER BY table_type, table_name`)
	defer rows.Close()
	assert.NoError(t, err)

	var count int
	for rows.Next() {
		count++
	}

	assert.Equal(t, 1, count)
}
