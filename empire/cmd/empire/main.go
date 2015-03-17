package main

import (
	"os"
	"path"

	"github.com/codegangsta/cli"
	"github.com/fsouza/go-dockerclient"
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
				Name:   "port",
				Value:  "8080",
				Usage:  "The port to run the server on",
				EnvVar: "PORT",
			},
			cli.StringFlag{
				Name:   "github.client.id",
				Value:  "",
				Usage:  "The client id for the GitHub OAuth application",
				EnvVar: "GITHUB_CLIENT_ID",
			},
			cli.StringFlag{
				Name:   "github.client.secret",
				Value:  "",
				Usage:  "The client secret for the GitHub OAuth application",
				EnvVar: "GITHUB_CLIENT_SECRET",
			},
			cli.StringFlag{
				Name:   "github.organization",
				Value:  "",
				Usage:  "The organization to allow access to",
				EnvVar: "GITHUB_ORGANIZATION",
			},
			cli.StringFlag{
				Name:   "github.secret",
				Value:  "",
				Usage:  "The shared secret between GitHub and Empire. GitHub will use this secret to sign webhook requests.",
				EnvVar: "GITHUB_SECRET",
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
		Name:   "db",
		Value:  "postgres://localhost/empire?sslmode=disable",
		Usage:  "SQL connection string for the database",
		EnvVar: "DATABASE_URL",
	},
}

var EmpireFlags = []cli.Flag{
	cli.StringFlag{
		Name:   "docker.organization",
		Value:  "",
		Usage:  "The fallback docker registry organization to use when an app is not linked to a docker repo. (e.g. quay.io/remind101)",
		EnvVar: "DOCKER_ORGANIZATION",
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
	cli.StringFlag{
		Name:   "fleet.api",
		Value:  "",
		Usage:  "The location of the fleet api",
		EnvVar: "FLEET_URL",
	},
	cli.StringFlag{
		Name:   "secret",
		Value:  "<change this>",
		Usage:  "The secret used to sign access tokens",
		EnvVar: "TOKEN_SECRET",
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
func empireOptions(c *cli.Context) (empire.Options, error) {
	opts := empire.Options{}

	opts.Docker.Organization = c.String("docker.organization")
	opts.Docker.Socket = c.String("docker.socket")
	opts.Docker.CertPath = c.String("docker.cert")
	opts.Fleet.API = c.String("fleet.api")
	opts.DB = c.String("db")
	opts.Secret = c.String("secret")

	auth, err := dockerAuth(c.String("docker.auth"))
	if err != nil {
		return opts, err
	}

	opts.Docker.Auth = auth

	return opts, nil
}

func dockerAuth(path string) (*docker.AuthConfigurations, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	return docker.NewAuthConfigurations(f)
}
