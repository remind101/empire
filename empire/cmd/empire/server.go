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
	opts, err := empireOptions(c)
	if err != nil {
		log.Fatal(err)
	}

	e, err := empire.New(opts)
	if err != nil {
		log.Fatal(err)
	}

	sopts := server.Options{}
	sopts.GitHub.ClientID = c.String("github.client.id")
	sopts.GitHub.ClientSecret = c.String("github.client.secret")
	sopts.GitHub.Organization = c.String("github.organization")
	s := server.New(e, sopts)

	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}
