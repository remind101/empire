package main

import (
	"io"
	"os"

	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/term"
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
	r, w := io.Pipe()

	image := c.Args[0]
	form := &PostDeployForm{Image: image}

	go func() {
		plugin.Must(c.Client.Post(w, "/deploys", form))
		plugin.Must(w.Close())
	}()

	outFd, isTerminalOut := term.GetFdInfo(os.Stdout)
	plugin.Must(jsonmessage.DisplayJSONMessagesStream(r, os.Stdout, outFd, isTerminalOut))
}
