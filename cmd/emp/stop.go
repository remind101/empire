package main

import (
	"log"
	"os"
)

var cmdStop = &Command{
	Run:             maybeMessage(runStop),
	Usage:           "stop <id>",
	NeedsApp:        true,
	OptionalMessage: true,
	Category:        "dyno",
	Short:           "stop processes",
	Long: `
Stops a given process by its id. The id of the process can be found in the output of ` + "`emp ps`" + `

Examples:

    $ emp stop 56bece28-f2fd-47b5-9f39-fbeaaf0d6fea -a <app>
    Stopped ` + "`56bece28-f2fd-47b5-9f39-fbeaaf0d6fe`" + `
`,
}

func runStop(cmd *Command, args []string) {
	appname := mustApp()
	if len(args) > 2 || len(args) < 1 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	message := getMessage()

	target := args[0]
	must(client.DynoRestart(appname, target, message))

	log.Printf("Stopped `%s`.", target)
}
