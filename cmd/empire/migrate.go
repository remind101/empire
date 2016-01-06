package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
)

func runMigrate(c *cli.Context) {
	db, err := newDB(c)
	if err != nil {
		log.Fatal(err)
	}

	path := c.String(FlagDBPath)
	errors, ok := db.MigrateUp(path)
	if !ok {
		log.Fatal(errors)
	}

	fmt.Println("Up to date")
}
