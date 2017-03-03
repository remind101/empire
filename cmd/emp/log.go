package main

import (
	"fmt"
	"os"
	"time"
)

var duration string

var cmdLog = &Command{
	Run:      runLog,
	Usage:    "log [-d]",
	NeedsApp: true,
	Category: "app",
	NumArgs:  0,
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
	Duration int64 `json:"duration"`
}

func runLog(cmd *Command, args []string) {
	cmd.AssertNumArgsCorrect(args)

	var d int64
	if duration != "" {
		parsed, err := time.ParseDuration(duration)
		if err != nil {
			fmt.Println(err)
			cmd.PrintUsage()
			os.Exit(1)
		}
		d = parsed.Nanoseconds()
	}

	appName := mustApp()
	endpoint := fmt.Sprintf("/apps/%s/log-sessions", appName)
	form := &PostLogForm{Duration: d}

	must(client.Post(os.Stdout, endpoint, form))
}
