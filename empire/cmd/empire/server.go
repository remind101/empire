package main

import (
	"log"
	"net/http"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server"
)

func runServer(c *cli.Context) {
	port := c.String("port")

	if c.Bool("automigrate") {
		runMigrate(c)
	}

	e, err := newEmpire(c)
	if err != nil {
		log.Fatal(err)
	}

	s := newServer(c, e)
	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}

func newServer(c *cli.Context, e *empire.Empire) http.Handler {
	opts := server.Options{}
	opts.GitHub.ClientID = c.String("github.client.id")
	opts.GitHub.ClientSecret = c.String("github.client.secret")
	opts.GitHub.Organization = c.String("github.organization")

	return server.New(e, opts)
}
