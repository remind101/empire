package main

import (
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire/empire"
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
		}, append(EmpireFlags, DBFlags...)...),
		Action: runServer,
	},
	{
		Name:  "migrate",
		Usage: "Migrate the database",
		Flags: append([]cli.Flag{
			cli.StringFlag{
				Name:  "path",
				Value: "./migrations",
				Usage: "Path to database migrations",
			},
		}, DBFlags...),
		Action: runMigrate,
	},
}

var DBFlags = []cli.Flag{
	cli.StringFlag{
		Name:  "db",
		Value: "postgres://localhost/empire?sslmode=disable",
		Usage: "SQL connection string for the database",
	},
}

var EmpireFlags = []cli.Flag{
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
	cli.StringFlag{
		Name:   "fleet.api",
		Value:  "",
		Usage:  "The location of the fleet api",
		EnvVar: "FLEET_URL",
	},
	cli.StringFlag{
		Name:   "github.secret",
		Value:  "",
		Usage:  "The shared secret for GitHub webhooks",
		EnvVar: "GITHUB_SECRET",
	},
	cli.StringFlag{
		Name:   "github.token",
		Value:  "",
		Usage:  "The github oauth token to use to create deployment statuses.",
		EnvVar: "GITHUB_TOKEN",
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
	opts.Docker.CertPath = c.String("docker.cert")
	opts.Docker.AuthPath = c.String("docker.auth")
	opts.Fleet.API = c.String("fleet.api")
	opts.DB = c.String("db")
	opts.GitHub.Secret = c.String("github.secret")
	opts.GitHub.Token = c.String("github.token")

	return opts
}
