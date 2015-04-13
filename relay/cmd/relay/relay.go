package main

import (
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire/relay"

	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
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
	l := logger.New(log.New(os.Stdout, "", 0))
	ctx := context.Background()
	ctx = logger.WithLogger(ctx, l)

	httpPort := c.String("http.port")
	tcpPort := c.String("tcp.port")

	go relay.ListenAndServeTCP(ctx, r, tcpPort)

	s := relay.NewHTTPServer(ctx, r)
	log.Printf("Starting http server on port %s\n", httpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, s))
}

func newRelay(c *cli.Context) *relay.Relay {
	o := relay.Options{}
	return relay.New(o)
}
