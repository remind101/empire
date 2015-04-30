package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"

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
	noStream := c.Flags.Bool("no-stream", false, "If true, output will not be streamed to the terminal.")
	c.Flags.Parse(c.Args[1:])

	if len(c.Args) < 1 {
		fmt.Println("Usage: emp deploy repo:id")
		return
	}

	var w io.Writer
	if *noStream {
		w = ioutil.Discard
	} else {
		w = os.Stdout
	}

	image := c.Args[0]
	form := &PostDeployForm{Image: image}

	err := c.Client.Post(w, "/deploys", form)
	if err != nil {
		plugin.Must(err)
	}

	fmt.Printf("Deployed %s\n", image)
}
