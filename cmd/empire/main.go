package main

import (
	"flag"
	"log"
	"net/http"
	"os"

	"github.com/remind101/empire"
)

func main() {
	opts := empire.Options{}

	var (
		port = flag.String("port", "8080", "The port to run the API on.")
	)

	flag.StringVar(&opts.Docker.Socket, "docker.socket", os.Getenv("DOCKER_HOST"), "The docker socket to connect to the docker api. Leave blank to use a fake extractor.")
	flag.StringVar(&opts.Docker.Registry, "docker.registry", "", "The docker registry to pull container images from. Leave blank to use the official docker registry.")
	flag.StringVar(&opts.Docker.CertPath, "docker.cert", os.Getenv("DOCKER_CERT_PATH"), "Path to certificate to use for TLS.")
	flag.StringVar(&opts.Fleet.API, "fleet.api", "http://127.0.0.1:49153", "The location of the fleet api.")

	flag.Parse()

	e, err := empire.New(opts)
	if err != nil {
		log.Fatal(err)
	}

	s := empire.NewServer(e)

	log.Printf("Starting on port %s", *port)
	log.Fatal(http.ListenAndServe(":"+*port, s))
}
