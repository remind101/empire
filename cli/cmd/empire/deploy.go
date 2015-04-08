package main

import (
	"fmt"

	"github.com/remind101/empire/cli/pkg/plugin"
)

var pluginDeploy = plugin.Plugin{
	Name:   "deploy",
	Action: runDeploy,
}

func runDeploy(c *plugin.Context) {
	fmt.Println(c.Client)
}
