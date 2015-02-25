package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
)

func runMigrate(c *cli.Context) {
	path := c.String("path")
	db := c.String("db")

	errors, ok := empire.Migrate(db, path)
	if !ok {
		log.Fatal(errors)
	}

	fmt.Println("Up to date")
}
