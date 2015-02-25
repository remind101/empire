package main

import (
	"log"
	"net/http"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
)

func runServer(c *cli.Context) {
	port := c.String("port")
	opts := empireOptions(c)

	e, err := empire.New(opts)
	if err != nil {
		log.Fatal(err)
	}

	s := empire.NewServer(e)

	log.Printf("Starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, s))
}
