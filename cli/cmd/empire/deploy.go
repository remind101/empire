package main

import (
	"fmt"
	"strings"

	"github.com/remind101/empire/cli/pkg/plugin"
)

var pluginDeploy = plugin.Plugin{
	Name:   "deploy",
	Action: runDeploy,
}

type Image struct {
	Repo string `json:"repo"`
	ID   string `json:"id"`
}

type PostDeployForm struct {
	Image *Image `json:"image"`
}

func runDeploy(c *plugin.Context) {
	if len(c.Args) < 1 {
		printUsage()
		return
	}

	parts := strings.Split(c.Args[0], ":")
	if len(parts) < 2 {
		printUsage()
		return
	}

	repo, id := parts[0], parts[1]
	form := &PostDeployForm{&Image{repo, id}}

	err := c.Client.Post(nil, "/deploys", form)
	if err != nil {
		plugin.Must(err)
	}

	fmt.Printf("Deployed %s:%s\n", repo, id)
}

func printUsage() {
	fmt.Println("Usage: emp deploy repo:id")
}
