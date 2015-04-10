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

type ImageParts struct {
	Repo string `json:"repo"`
	ID   string `json:"id"`
}

type Image struct {
	Image *ImageParts `json:"image"`
}

func runDeploy(c *plugin.Context) {
	repo, id := strings.Split(c.Args[0], ":")[0], strings.Split(c.Args[0], ":")[1]
	image := &Image{&ImageParts{repo, id}}

	err := c.Client.Post(nil, "/deploys", image)
	if err != nil {
		fmt.Printf("Failed to deploy %s:%s\n", repo, id)
		plugin.Must(err)
	}

	fmt.Printf("Deployed %s:%s\n", repo, id)
}
