package main

import (
	"fmt"
	"os"
)

var cmdLog = &Command{
	Run:      runLog,
	Usage:    "log",
	NeedsApp: true,
	Category: "app",
	Short:    "stream app log lines",
	Long: `
Log prints the streaming application log.
   Examples:
    $ emp log -a acme-inc
    2013-10-17T00:17:35.066089+00:00 app[web.1]: Completed 302 Found in 0ms
    ...
`,
}

func runLog(cmd *Command, args []string) {
	if len(args) != 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}

	appName := mustApp()
	endpoint := fmt.Sprintf("/apps/%s/log-sessions", appName)

	must(client.Post(os.Stdout, endpoint, nil))
}
