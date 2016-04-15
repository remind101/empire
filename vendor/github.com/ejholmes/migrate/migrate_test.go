package migrate_test

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/ejholmes/migrate"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

var testMigrations = []migrate.Migration{
	{
		ID: 1,
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec("CREATE TABLE people (id int)")
			return err
		},
		Down: func(tx *sql.Tx) error {
			_, err := tx.Exec("DROP TABLE people")
			return err
		},
	},
	{
		ID: 2,
		// For simple sql migrations, you can use the migrate.Queries
		// helper.
		Up: migrate.Queries([]string{
			"ALTER TABLE people ADD COLUMN first_name text",
		}),
		Down: func(tx *sql.Tx) error {
			// It's not possible to remove a column with
			// sqlite.
			_, err := tx.Exec("SELECT 1 FROM people")
			return err
		},
	},
}

func TestMigrate(t *testing.T) {
	db := newDB(t)

	migrations := testMigrations[:]

	err := migrate.Exec(db, migrate.Up, migrations...)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, appliedMigrations(t, db))
	assertSchema(t, `
people
CREATE TABLE people (id int, first_name text)
`, db)

	err = migrate.Exec(db, migrate.Down, migrations...)
	assert.NoError(t, err)
	assert.Equal(t, []int{}, appliedMigrations(t, db))
	assertSchema(t, ``, db)
}

func TestMigrate_Individual(t *testing.T) {
	db := newDB(t)

	err := migrate.Exec(db, migrate.Up, testMigrations[0])
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, appliedMigrations(t, db))
	assertSchema(t, `
people
CREATE TABLE people (id int)
`, db)

	err = migrate.Exec(db, migrate.Up, testMigrations[1])
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, appliedMigrations(t, db))
	assertSchema(t, `
people
CREATE TABLE people (id int, first_name text)
`, db)
}

func TestMigrate_AlreadyRan(t *testing.T) {
	db := newDB(t)

	migration := testMigrations[0]

	err := migrate.Exec(db, migrate.Up, migration)
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, appliedMigrations(t, db))
	assertSchema(t, `
people
CREATE TABLE people (id int)
`, db)

	err = migrate.Exec(db, migrate.Up, migration)
	assert.NoError(t, err)
	assert.Equal(t, []int{1}, appliedMigrations(t, db))
	assertSchema(t, `
people
CREATE TABLE people (id int)
`, db)
}

func TestMigrate_Order(t *testing.T) {
	db := newDB(t)

	migrations := []migrate.Migration{
		testMigrations[1],
		testMigrations[0],
	}

	err := migrate.Exec(db, migrate.Up, migrations...)
	assert.NoError(t, err)
	assert.Equal(t, []int{1, 2}, appliedMigrations(t, db))
	assertSchema(t, `
people
CREATE TABLE people (id int, first_name text)
`, db)
}

func TestMigrate_Rollback(t *testing.T) {
	db := newDB(t)

	migration := migrate.Migration{
		ID: 1,
		Up: func(tx *sql.Tx) error {
			// This should completely ok
			if _, err := tx.Exec("CREATE TABLE people (id int)"); err != nil {
				return err
			}
			// This should throw an error
			if _, err := tx.Exec("ALTER TABLE foo ADD COLUMN first_name text"); err != nil {
				return err
			}
			return nil
		},
	}

	err := migrate.Exec(db, migrate.Up, migration)
	assert.Error(t, err)
	assert.Equal(t, []int{}, appliedMigrations(t, db))
	// If the transaction wasn't rolled back, we'd see a people table.
	assertSchema(t, ``, db)
	assert.IsType(t, &migrate.MigrationError{}, err)
}

func assertSchema(t testing.TB, expectedSchema string, db *sql.DB) {
	schema, err := schema(db)
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, strings.TrimSpace(expectedSchema), schema)
}

func schema(db *sql.DB) (string, error) {
	var tables []string
	rows, err := db.Query(`SELECT name, sql FROM sqlite_master
WHERE type='table'
ORDER BY name;`)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	for rows.Next() {
		var name, sql string
		if err := rows.Scan(&name, &sql); err != nil {
			return "", err
		}
		if name == migrate.DefaultTable {
			continue
		}
		tables = append(tables, fmt.Sprintf("%s\n%s", name, sql))
	}
	return strings.Join(tables, "\n\n"), nil
}

func appliedMigrations(t testing.TB, db *sql.DB) []int {
	rows, err := db.Query("SELECT version FROM " + migrate.DefaultTable)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
	}

	return ids
}

func newDB(t testing.TB) *sql.DB {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	return db
}
