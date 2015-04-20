package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
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
			cli.StringFlag{
				Name:   "docker.socket",
				Value:  "unix:///var/run/docker.sock",
				Usage:  "The location of the docker api",
				EnvVar: "DOCKER_HOST",
			},
			cli.StringFlag{
				Name:   "docker.cert",
				Value:  "",
				Usage:  "If using TLS, a path to a certificate to use",
				EnvVar: "DOCKER_CERT_PATH",
			},
			cli.StringFlag{
				Name:   "docker.auth",
				Value:  path.Join(os.Getenv("HOME"), ".dockercfg"),
				Usage:  "Path to a docker registry auth file (~/.dockercfg)",
				EnvVar: "DOCKER_AUTH_PATH",
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

	cert, err := tls.LoadX509KeyPair("rendezvous.172.20.20.10.xip.io.crt", "rendezvous.172.20.20.10.xip.io.key")
	if err != nil {
		panic(err)
	}

	tcpPort := c.String("tcp.port")
	log.Printf("Starting tcp server on port %s\n", tcpPort)
	l, err := tls.Listen("tcp", ":"+tcpPort, &tls.Config{
		Certificates: []tls.Certificate{cert},
	})

	// Start TCP Server in a go routine
	th := relay.NewTCPHandler(r)
	go tcp.Serve(l, th)

	// Start HTTP Server
	hh := relay.NewHTTPHandler(r)
	httpPort := c.String("http.port")
	log.Printf("Starting http server on port %s\n", httpPort)
	log.Fatal(http.ListenAndServe(":"+httpPort, hh))
}

func newRelay(c *cli.Context) *relay.Relay {
	opts := relay.Options{}

	opts.Docker.Socket = c.String("docker.socket")
	opts.Docker.CertPath = c.String("docker.cert")

	auth, err := dockerAuth(c.String("docker.auth"))
	if err != nil {
		panic(err)
	}

	opts.Docker.Auth = auth
	opts.Tcp.Host = c.String("tcp.host")
	opts.Tcp.Port = c.String("tcp.port")

	return relay.New(opts)
}

func dockerAuth(path string) (*docker.AuthConfigurations, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return docker.NewAuthConfigurations(f)
}
