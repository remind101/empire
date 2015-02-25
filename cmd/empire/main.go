package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
)

// Commands are the subcommands that are available.
var Commands = []cli.Command{
	{
		Name:      "server",
		ShortName: "s",
		Usage:     "Run the empire HTTP api",
		Flags: append([]cli.Flag{
			cli.StringFlag{
				Name:  "port",
				Value: "8080",
				Usage: "The port to run the server on",
			},
		}, EmpireFlags...),
		Action: runServer,
	},
}

// Flags related to empire.Options.
var EmpireFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "docker.socket",
		Value:  "unix:///var/run/docker.sock",
		Usage:  "The location of the docker api",
		EnvVar: "DOCKER_HOST",
	},
	cli.StringFlag{
		Name:   "docker.registry",
		Value:  "",
		Usage:  "The docker registry to pull container images from",
		EnvVar: "DOCKER_HOST",
	},
	cli.StringFlag{
		Name:   "docker.cert",
		Value:  "",
		Usage:  "If using TLS, a path to a certificate to use",
		EnvVar: "DOCKER_CERT_PATH",
	},
	cli.StringFlag{
		Name:  "fleet.api",
		Value: "http://127.0.0.1:49153",
		Usage: "The location of the fleet api",
	},
}

func main() {
	app := cli.NewApp()
	app.Name = "empire"
	app.Usage = "Platform as a Binary"
	app.Commands = Commands

	app.Run(os.Args)
}

// Returns a new empire.Options based on provided flags.
func empireOptions(c *cli.Context) empire.Options {
	opts := empire.Options{}

	opts.Docker.Socket = c.String("docker.socket")
	opts.Docker.Registry = c.String("docker.registry")
	opts.Docker.CertPath = c.String("docker.cert")
	opts.Fleet.API = c.String("fleet.api")

	return opts
}
