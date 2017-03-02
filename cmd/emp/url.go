package main

import "fmt"

var cmdURL = &Command{
	Run:      runURL,
	Usage:    "url",
	NeedsApp: true,
	Category: "app",
	NumArgs:  0,
	Short:    "show app url" + extra,
	Long:     `Prints the web URL for the app.`,
}

func runURL(cmd *Command, args []string) {
	cmd.CheckNumArgs(args)

	app, err := client.AppInfo(mustApp())
	must(err)
	fmt.Println(app.WebURL)
}
