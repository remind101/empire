package main

import (
	"fmt"
	"os"
)

var duration string

var cmdLog = &Command{
	Run:      runLog,
	Usage:    "log [-d]",
	NeedsApp: true,
	Category: "app",
	Short:    "stream app log lines",
	Long: `
Log prints the streaming application log.

Options:

	-d duration to go back and start reading logs from (ie. 10m will start
	   streaming from 10 minutes ago)

Examples:

	$ emp log -a acme-inc
	2013-10-17T00:17:35.066089+00:00 app[web.1]: Completed 302 Found in 0ms
	...
`,
}

func init() {
	cmdLog.Flag.StringVarP(&duration, "duration", "d", "", "duration to start streaming logs from")
}

type PostLogForm struct {
	Duration string `json:"duration"`
}

func runLog(cmd *Command, args []string) {
	if len(args) != 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}

	appName := mustApp()
	endpoint := fmt.Sprintf("/apps/%s/log-sessions", appName)
	form := &PostLogForm{Duration: duration}

	must(client.Post(os.Stdout, endpoint, form))
}
