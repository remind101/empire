package empire_test

import (
	"fmt"

	"github.com/remind101/empire"
)

func Example() {
	// Open a postgres connection.
	db, _ := empire.OpenDB("postgres://localhost/empire?sslmode=disable")

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
