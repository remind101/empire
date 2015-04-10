package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire/relay"
	"github.com/remind101/pkg/httpx"
	"github.com/remind101/pkg/httpx/middleware"
	"github.com/remind101/pkg/reporter"
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
		Action: runServer,
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "relay"
	app.Usage = "A heroku compatible rendezvous server"
	app.Commands = Commands

	app.Run(os.Args)
}

func runServer(c *cli.Context) {
	httpPort := c.String("http.port")
	tcpPort := c.String("tcp.port")

	r := newRelay(c)
	s := newServer(r)

	log.Printf("Starting on ports http:%s tcp:%s", httpPort, tcpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, s))
}

func newRelay(c *cli.Context) *relay.Relay {
	o := relay.Options{}
	return relay.New(o)
}

func newServer(r *relay.Relay) http.Handler {
	m := httpx.NewRouter()

	m.Handle("GET", "/containers", &PostContainers{r})

	var h httpx.Handler

	// Recover from panics.
	h = middleware.Recover(m, reporter.NewLogReporter())

	// Add a logger to the context.
	h = middleware.NewLogger(h, os.Stdout)

	// Add the request id to the context.
	h = middleware.ExtractRequestID(h)

	// Wrap the route in middleware to add a context.Context.
	return middleware.BackgroundContext(h)
}

type PostContainers struct {
	*relay.Relay
}

func (h *PostContainers) ServeHTTPContext(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	return Encode(w, nil)
}

func Encode(w http.ResponseWriter, v interface{}) error {
	if v == nil {
		// Empty JSON body "{}"
		v = map[string]interface{}{}
	}

	return json.NewEncoder(w).Encode(v)
}
