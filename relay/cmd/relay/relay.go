package main

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire/relay"
	"github.com/remind101/empire/relay/tcp"
)

var Commands = []cli.Command{
	{
		Name:      "server",
		ShortName: "s",
		Usage:     "Run the relay server",
		Flags: []cli.Flag{
			cli.StringFlag{
				Name:   "http.port",
				Value:  "9000",
				Usage:  "The port to run the http server on",
				EnvVar: "RELAY_HTTP_PORT",
			},
			cli.StringFlag{
				Name:   "tcp.host",
				Value:  "",
				Usage:  "The hostname the tcp server is running on.",
				EnvVar: "RELAY_TCP_HOST",
			},
			cli.StringFlag{
				Name:   "tcp.port",
				Value:  "5000",
				Usage:  "The port to run the tcp server on",
				EnvVar: "RELAY_TCP_PORT",
			},
		},
		Action: runServers,
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "relay"
	app.Usage = "A heroku compatible rendezvous server"
	app.Commands = Commands

	app.Run(os.Args)
}

func runServers(c *cli.Context) {
	r := newRelay(c)

	// Start TCP Server in a go routine
	th := relay.NewTCPHandler(r)
	tcpPort := c.String("tcp.port")
	log.Printf("Starting tcp server on port %s\n", tcpPort)
	go tcp.ListenAndServe(":"+tcpPort, th)

	// Start HTTP Server
	hh := relay.NewHTTPHandler(r)
	httpPort := c.String("http.port")
	log.Printf("Starting http server on port %s\n", httpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, hh))
}

func newRelay(c *cli.Context) *relay.Relay {
	return relay.New(relay.DefaultOptions)
}
