package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/codegangsta/cli"
)

const target = "current"

func setEnv() {
	EMPIRE_URL := os.Getenv("EMPIRE_URL")
	if EMPIRE_URL == "" {
		EMPIRE_URL = config[config[target]]
	}
	os.Setenv("HEROKU_API_URL", EMPIRE_URL)
}

var cmdListAPIs = cli.Command{
	Name:   "apis",
	Usage:  "List the Empire APIs",
	Action: listAPIs,
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

var cmdAddAPI = cli.Command{
	Name:   "api-add",
	Usage:  "Add one or several API targets.",
	Action: addAPI,
}

func addAPI(c *cli.Context) {
	if len(c.Args()) > 0 {
		for _, arg := range c.Args() {
			api := strings.Split(arg, "=")
			config[api[0]] = api[1]
			configOrder = append(configOrder, api[0])
		}
		saveConfig()
		fmt.Println("Added api target(s)")
	}
}

var cmdSetAPI = cli.Command{
	Name:   "api-set",
	Usage:  "Set the API target.",
	Action: setAPI,
}

func setAPI(c *cli.Context) {
	if len(c.Args()) > 0 {
		config[target] = c.Args()[0]
		if newTarget, ok := config[config[target]]; ok {
			setEnv()
			saveConfig()
			fmt.Printf("emp now pointed at %s (%s)\n", config[target], newTarget)
		} else {
			fmt.Println("You need to add this API first with `emp api-add <target>`")
		}
	}
}
