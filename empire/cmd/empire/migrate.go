package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire/empire"
)

func runMigrate(c *cli.Context) {
	path := c.String(FlagDBPath)
	db := c.String(FlagDB)

	errors, ok := empire.Migrate(db, path)
	if !ok {
		log.Fatal(errors)
	}

	fmt.Println("Up to date")
}
