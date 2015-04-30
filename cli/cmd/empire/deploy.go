package main

import (
	"fmt"

	"github.com/remind101/empire/cli/pkg/plugin"
)

var pluginDeploy = plugin.Plugin{
	Name:   "deploy",
	Action: runDeploy,
}

type PostDeployForm struct {
	Image string `json:"image"`
}

func runDeploy(c *plugin.Context) {
	if len(c.Args) < 1 {
		fmt.Println("Usage: emp deploy repo:id")
		return
	}

	image := c.Args[0]
	form := &PostDeployForm{Image: image}

	err := c.Client.Post(nil, "/deploys", form)
	if err != nil {
		plugin.Must(err)
	}

	fmt.Printf("Deployed %s\n", image)
}
