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

	if err := db.MigrateUp(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Up to date")
}
