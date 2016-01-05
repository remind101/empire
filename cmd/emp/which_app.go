package main

import (
	"fmt"
)

var cmdWhichApp = &Command{
	Run:      runWhichApp,
	Usage:    "which-app",
	NeedsApp: true,
	Category: "app",
	Short:    "show which app is selected, if any" + extra,
	Long: `
Looks for a git remote named "heroku" with a remote URL in the
correct form. If successful, it prints the corresponding app name.
Otherwise, it prints an error message to stderr and exits with a
nonzero status.

To suppress the error message, run 'emp app 2>/dev/null'.
`,
}

func runWhichApp(cmd *Command, args []string) {
	fmt.Println(mustApp())
}
