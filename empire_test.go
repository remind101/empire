package empire_test

import (
	"fmt"

	"github.com/remind101/empire"
	"github.com/remind101/empire/dbtest"
)

func Example() {
	// Open a postgres connection.
	db, _ := empire.OpenDB(*dbtest.DatabaseURL)

	// Migrate the database schema.
	_ = db.MigrateUp()

	// Initialize a new Empire instance.
	e := empire.New(db)

	// Run operations against Empire.
	apps, _ := e.Apps(empire.AppsQuery{})
	fmt.Println(apps)
	// Output:
	// []
}
