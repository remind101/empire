package main

import (
	"fmt"
	"log"
	"os/exec"
)

var cmdDestroy = &Command{
	Run:             confirmDestroy(maybeMessage(runDestroy)),
	Usage:           "destroy <name>",
	OptionalMessage: true,
	Category:        "app",
	Short:           "destroy an app",
	NumArgs:         1,
	Long: `
Destroy destroys a heroku app. There is no going back, so be
sure you mean it. The command will prompt for confirmation, or
accept confirmation via stdin.

Example:

    $ emp destroy myapp
    warning: This will destroy myapp and its add-ons. Please type "myapp" to continue:
    Destroyed myapp.

    $ echo myapp | emp destroy myapp
    Destroyed myapp.
`,
}

func confirmDestroy(action func(cmd *Command, args []string)) func(cmd *Command, args []string) {
	return func(cmd *Command, args []string) {
		if len(args) != 1 {
			cmd.PrintUsage()
			os.Exit(2)
		}

		appname := args[0]
		warning := fmt.Sprintf("This will destroy %s and its add-ons. Please type %q to continue:", appname, appname)
		mustConfirm(warning, appname)
		action(cmd, args)
	}
}

func runDestroy(cmd *Command, args []string) {
	message := getMessage()
	appname := args[0]

	must(client.AppDelete(appname, message))
	log.Printf("Destroyed %s.", appname)
	remotes, _ := gitRemotes()
	for remote, remoteApp := range remotes {
		if appname == remoteApp {
			exec.Command("git", "remote", "rm", remote).Run()
		}
	}
}
