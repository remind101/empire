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
	Usage:  "List the Empire apis",
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
	Usage:  "Add one or several api targets",
	Action: addAPI,
}

func addAPI(c *cli.Context) {
	if len(c.Args()) > 0 {
		for _, arg := range c.Args() {
			api := strings.Split(arg, "=")

			if _, present := config[api[0]]; !present {
				config[api[0]] = api[1]
				configOrder = append(configOrder, api[0])
				saveConfig()
				fmt.Println("Added api target(s)")
			} else {
				fmt.Printf("Can't add api target %s, already present in config\n", api[0])
				continue
			}
		}
	}
}

var cmdSetAPI = cli.Command{
	Name:   "api-set",
	Usage:  "Set the api target",
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
			fmt.Println("You need to add this api target first with `emp api-add <target>`")
		}
	}
}

var cmdDeleteAPI = cli.Command{
	Name:   "api-delete",
	Usage:  "Delete the API target.",
	Action: deleteAPI,
}

func deleteAPI(c *cli.Context) {
	if len(c.Args()) > 0 {
		api := c.Args()[0]
		if api == config[target] {
			fmt.Println("You can't delete the current api target")
			return
		}

		delete(config, api)
		deleteOrder(api)
		saveConfig()
		fmt.Printf("Deleted api %s\n", api)
	}
}
