package main

import (
	"encoding/json"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/remind101/empire/pkg/heroku"
)

var cmdDynos = &Command{
	Run:      runDynos,
	Usage:    "ps",
	Alias:    "dynos",
	NeedsApp: true,
	Category: "dyno",
	Short:    "list processes",
	Long: `
Lists processes. Shows the name, size, id, container id, ec2 instance id, state, age, and command.

Examples:

    $ emp ps
    v1.run.e97e1f75e8ff                             2X  RUNNING   1m  bash
    v1.web.dcc9a8c4-c0f8-4478-aa8a-f9148b362401     1X  RUNNING  15h  "blog /app /tmp/dst"
    v1.web.2bcb6e08-ef99-447f-8e7a-416d94769010     1X  RUNNING   8h  "blog /app /tmp/dst"
`,
}

func runDynos(cmd *Command, args []string) {
	w := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	defer w.Flush()
	if len(args) != 0 {
		cmd.PrintUsage()
		os.Exit(2)
	}

	listDynos(w)
}

func listDynos(w io.Writer) {
	appname := mustApp()
	dynos, err := client.DynoList(appname, nil)
	must(err)
	sort.Sort(DynosByName(dynos))

	for _, d := range dynos {
		listDyno(w, &d)
	}
	return
}

func listDyno(w io.Writer, d *heroku.Dyno) {
	listRec(w,
		d.Name,
		d.Id,
		d.ContainerInstanceID,
		d.EC2InstanceID,
		d.Size,
		d.State,
		prettyDuration{dynoAge(d)},
		maybeQuote(d.Command),
	)
}

// quotes s as a json string if it contains any weird chars
// currently weird is anything other than [alnum]_-
func maybeQuote(s string) string {
	for _, r := range s {
		if !('0' <= r && r <= '9' || 'a' <= r && r <= 'z' ||
			'A' <= r && r <= 'Z' || r == '-' || r == '_') {
			return quote(s)
		}
	}
	return s
}

// quotes s as a json string
func quote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

type DynosByName []heroku.Dyno

func (p DynosByName) Len() int      { return len(p) }
func (p DynosByName) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
func (p DynosByName) Less(i, j int) bool {
	return p[i].Type < p[j].Type || p[i].Type == p[j].Type && dynoSeq(&p[i]) < dynoSeq(&p[j])
}

func dynoAge(d *heroku.Dyno) time.Duration {
	return time.Now().Sub(d.UpdatedAt)
}

func dynoSeq(d *heroku.Dyno) int {
	i, _ := strconv.Atoi(strings.TrimPrefix(d.Name, d.Type+"."))
	return i
}
