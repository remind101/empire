package main

import (
	"flag"
	"log"
	"net/http"

	"github.com/remind101/empire"
)

func main() {
	var (
		port = flag.String("port", "8080", "The port to run the API on.")
	)

	e := empire.New()
	s := empire.NewServer(e)

	log.Printf("Starting on port %s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, s))
}
