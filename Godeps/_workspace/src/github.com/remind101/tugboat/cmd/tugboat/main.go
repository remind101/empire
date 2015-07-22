package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/remind101/tugboat"
	"github.com/remind101/tugboat/provider/empire"
	"github.com/remind101/tugboat/provider/heroku"
)

var commands = []cli.Command{
	cmdServer,
	cmdMigrate,
	cmdTokens,
}

// Shared flags.
var (
	flagDB = cli.StringFlag{
		Name:   "db.url",
		Value:  "postgres://localhost/tugboat?sslmode=disable",
		Usage:  "Postgres connection string.",
		EnvVar: "DATABASE_URL",
	}
	flagProviderSecret = cli.StringFlag{
		Name:   "provider.secret",
		Value:  "",
		Usage:  "A secret used to sign provider tokens",
		EnvVar: "TUGBOAT_PROVIDER_SECRET",
	}
)

func main() {
	app := cli.NewApp()
	app.Name = "tugboat"
	app.Commands = commands

	app.Run(os.Args)
}

func newTugboat(c *cli.Context) (*tugboat.Tugboat, error) {
	config := tugboat.Config{}
	config.DB = c.String("db.url")
	config.Pusher.URL = c.String("pusher.url")
	config.GitHub.Token = c.String("github.token")
	config.ProviderSecret = []byte(c.String("provider.secret"))

	tug, err := tugboat.New(config)
	if err != nil {
		return tug, err
	}

	ps, err := newProviders(c)
	if err != nil {
		return tug, err
	}

	tug.Providers = ps
	tug.MatchEnvironment = c.String("environment")

	return tug, nil
}

func newProviders(c *cli.Context) ([]tugboat.Provider, error) {
	if c.String("provider") == "" {
		return nil, nil
	}

	uris := strings.Split(c.String("provider"), ",")

	var providers []tugboat.Provider

	for _, uri := range uris {
		p, err := newProvider(c, uri)
		if err != nil {
			return nil, err
		}

		providers = append(providers, p)
	}

	return providers, nil
}

func newProvider(c *cli.Context, uri string) (tugboat.Provider, error) {
	u, err := url.Parse(uri)
	if err != nil {
		return nil, err
	}

	switch u.Scheme {
	case "heroku":
		return heroku.NewProvider(
			c.String("github.token"),
			u.Query().Get("token"),
		), nil
	case "empire":
		return empire.NewProvider(
			fmt.Sprintf("https://%s", u.Host),
			u.Query().Get("token"),
		), nil
	default:
		return nil, fmt.Errorf("No provider matching %s", u)
	}
}
