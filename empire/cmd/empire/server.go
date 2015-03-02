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
	opts := empireOptions(c)

	e, err := empire.New(opts)
	if err != nil {
		log.Fatal(err)
	}

	s := server.New(e)

	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}
