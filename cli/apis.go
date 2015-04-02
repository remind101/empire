package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
)

const target = "current"

func setAPI() {
	EMPIRE_URL := os.Getenv("EMPIRE_URL")
	if EMPIRE_URL == "" {
		EMPIRE_URL = config[config[target]]
	}
	os.Setenv("HEROKU_API_URL", EMPIRE_URL)
}

var cmdListAPIs = cli.Command{
	Name:      "apis",
	ShortName: "apis",
	Usage:     "List the Empire APIs",
	Action:    listAPIs,
}

func listAPIs(c *cli.Context) {
	for _, key := range configOrder {
		if key == target {
			continue
		}
		if key == config[target] {
			fmt.Printf("* ")
		}
		fmt.Printf("%s \t %s\n", key, config[key])
	}
}
