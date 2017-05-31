package empire

import (
	"testing"

	_ "github.com/lib/pq"
	"github.com/remind101/empire/dbtest"
	"github.com/remind101/migrate"
	"github.com/stretchr/testify/assert"
)

// Tests migrating the database down, then back up again.
func TestMigrations(t *testing.T) {
	db, err := NewDB(dbtest.Open(t))
	if err != nil {
		t.Fatal(err)
	}

	err = db.migrator.Exec(migrate.Up, migrations...)
	assert.NoError(t, err)

	err = db.Reset()
	assert.NoError(t, err)

	err = db.migrator.Exec(migrate.Down, migrations...)
	assert.NoError(t, err)

	err = db.migrator.Exec(migrate.Up, migrations...)
	assert.NoError(t, err)
}

func TestLatestSchema(t *testing.T) {
	assert.Equal(t, 19, latestSchema())
}

func TestNoDuplicateMigrations(t *testing.T) {
	visited := make(map[int]bool)
	expectedID := 1
	for _, m := range migrations {
		if visited[m.ID] {
			t.Fatalf("Migration %d appears more than once", m.ID)
		}
		visited[m.ID] = true
		if m.ID != expectedID {
			t.Fatalf("Expected migration %d after %d, but got %d", expectedID, expectedID-1, m.ID)
		}
		expectedID++
	}
}
