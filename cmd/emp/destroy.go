package main

import (
	"fmt"
	"log"
	"os/exec"
)

var cmdDestroy = &Command{
	Run:             mustConfirmAndMessageRequired(runDestroy, warningMessage),
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

func runDestroy(cmd *Command, args []string) {
	cmd.AssertNumArgsCorrect(args)
	appname := args[0]
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

func warningMessage(args []string) (warning, desired string) {
	appname := args[0]
	warning = fmt.Sprintf("This will destroy %s and its add-ons. Please type %q to continue:", appname, appname)
	desired = appname
	return
}
