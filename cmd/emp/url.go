package main

import (
	"fmt"
	"os"
)

var cmdURL = &Command{
	Run:      runURL,
	Usage:    "url",
	NeedsApp: true,
	Category: "app",
	Short:    "show app url" + extra,
	Long:     `Prints the web URL for the app.`,
}

func runURL(cmd *Command, args []string) {
	if len(args) != 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	app, err := client.AppInfo(mustApp())
	must(err)
	fmt.Println(app.WebURL)
}
