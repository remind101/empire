package main

import (
	"log"
	"os"

	"github.com/remind101/empire/pkg/heroku"
)

var cmdRename = &Command{
	Run:      runRename,
	Usage:    "rename <oldname> <newname>",
	Category: "app",
	Short:    "rename an app",
	Long: `
Rename renames a heroku app.

Example:

    $ emp rename myapp myapp2
`,
}

func runRename(cmd *Command, args []string) {
	if len(args) != 2 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	oldname, newname := args[0], args[1]
	app, err := client.AppUpdate(oldname, &heroku.AppUpdateOpts{Name: &newname})
	must(err)
	log.Printf("Renamed %s to %s.", oldname, app.Name)
	log.Println("Ensure you update your git remote URL.")
	// should we automatically update the remote if they specify an app
	// or via mustApp + conditional logic - RM
}
