package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"text/template"

	"github.com/codegangsta/cli"
	"github.com/remind101/tugboat"
)

var inspectTemplate = `Provider: {{.Provider}}
Token: {{.Token}}`

var cmdTokens = cli.Command{
	Name: "tokens",
	Subcommands: []cli.Command{
		{
			Name:   "create",
			Usage:  "Creates a new token for an external provider. External providers should use this token as the basic auth user when creating deployments.",
			Action: runTokensCreate,
			Flags: []cli.Flag{
				flagDB,
				flagProviderSecret,
			},
		},
		{
			Name:   "inspect",
			Usage:  "Inspects a token",
			Action: runTokensInspect,
			Flags: []cli.Flag{
				flagDB,
				flagProviderSecret,
				cli.StringFlag{
					Name:  "template",
					Value: inspectTemplate,
				},
			},
		},
	},
}

func runTokensCreate(c *cli.Context) {
	tug, err := newTugboat(c)
	if err != nil {
		log.Fatal(err)
	}

	provider := c.Args().First()

	if provider == "" {
		fmt.Fprintf(os.Stderr, "You must specify a provider name.\n")
		os.Exit(-1)
	}

	token := &tugboat.Token{Provider: provider}
	if err := tug.TokensCreate(token); err != nil {
		log.Fatal(err)
	}

	fmt.Println(token.Token)
}

func runTokensInspect(c *cli.Context) {
	tug, err := newTugboat(c)
	if err != nil {
		log.Fatal(err)
	}

	token := c.Args().First()

	if token == "" {
		fmt.Fprintf(os.Stderr, "Please provide a token.")
		os.Exit(-1)
	}

	t, err := tug.TokensFind(token)
	if err != nil {
		log.Fatal(err)
	}

	tmpl := template.Must(template.New("token").Parse(c.String("template")))
	if err := tmpl.Execute(os.Stdout, t); err != nil {
		log.Fatal(err)
	}
	io.WriteString(os.Stdout, "\n")
}
