package main

import (
	"fmt"
	"log"

	"github.com/codegangsta/cli"
	"github.com/remind101/tugboat"
)

var cmdMigrate = cli.Command{
	Name:      "migrate",
	ShortName: "m",
	Usage:     "Migrate the database",
	Action:    runMigrate,
	Flags: []cli.Flag{
		flagDB,
		cli.StringFlag{
			Name:  "db.migrations",
			Value: "./migrations",
			Usage: "Path to the directory containing migrations.",
		},
	},
}

func runMigrate(c *cli.Context) {
	db := c.String("db.url")
	path := c.String("db.migrations")

	errors, ok := tugboat.Migrate(db, path)
	if !ok {
		log.Fatal(errors)
	}

	fmt.Println("Up to date")
}
