package main

import (
	"fmt"
	"log"

	"github.com/urfave/cli"
)

func runMigrate(c *cli.Context) {
	ctx, err := newContext(c)
	if err != nil {
		log.Fatal(err)
	}

	db, err := newDB(ctx)
	if err != nil {
		log.Fatal(err)
	}

	if err := db.MigrateUp(); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Up to date")
}
