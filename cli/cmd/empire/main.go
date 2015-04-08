package main

import (
	"log"
	"os"

	"github.com/remind101/empire/cli/pkg/plugin"
)

var plugins = []plugin.Plugin{
	pluginDeploy,
}

func main() {
	app := plugin.NewApp()
	app.Plugins = plugins

	if err := app.Run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}
