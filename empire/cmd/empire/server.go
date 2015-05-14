package main

import (
	"log"
	"net/http"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire/empire"
	"github.com/remind101/empire/empire/server"
)

func runServer(c *cli.Context) {
	port := c.String(FlagPort)

	if c.Bool(FlagAutoMigrate) {
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
	opts.GitHub.ClientID = c.String(FlagGithubClient)
	opts.GitHub.ClientSecret = c.String(FlagGithubSecret)
	opts.GitHub.Organization = c.String(FlagGithubOrg)

	return server.New(e, opts)
}
