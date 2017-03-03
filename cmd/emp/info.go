package main

import "fmt"

var cmdInfo = &Command{
	Run:      runInfo,
	Usage:    "info",
	NeedsApp: true,
	Category: "app",
	NumArgs:  0,
	Short:    "show app info",
	Long:     `Info shows general information about the current app.`,
}

func runInfo(cmd *Command, args []string) {
	cmd.AssertNumArgsCorrect(args)

	app, err := client.AppInfo(mustApp())
	must(err)
	fmt.Printf("Name: %s\n", app.Name)
	fmt.Printf("ID:   %s\n", app.Id)
	fmt.Printf("Cert: %s\n", app.Cert)
}
