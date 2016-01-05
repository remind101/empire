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
	Usage:    "ps [<name>...]",
	Alias:    "dynos",
	NeedsApp: true,
	Category: "dyno",
	Short:    "list processes",
	Long: `
Lists processes. Shows the name, size, state, age, and command.

Examples:

    $ emp ps
    run.3794  2X  up   1m  bash
    web.1     1X  up  15h  "blog /app /tmp/dst"
    web.2     1X  up   8h  "blog /app /tmp/dst"

    $ emp ps web
    web.1     1X  up  15h  "blog /app /tmp/dst"
    web.2     1X  up   8h  "blog /app /tmp/dst"
`,
}

func runDynos(cmd *Command, names []string) {
	w := tabwriter.NewWriter(os.Stdout, 1, 2, 2, ' ', 0)
	defer w.Flush()

	if len(names) > 1 {
		cmd.PrintUsage()
		os.Exit(2)
	}
	listDynos(w, names)
}

func listDynos(w io.Writer, names []string) {
	appname := mustApp()
	dynos, err := client.DynoList(appname, nil)
	must(err)
	sort.Sort(DynosByName(dynos))

	if len(names) == 0 {
		for _, d := range dynos {
			listDyno(w, &d)
		}
		return
	}

	for _, name := range names {
		for _, d := range dynos {
			if !strings.Contains(name, ".") {
				if strings.HasPrefix(d.Name, name+".") {
					listDyno(w, &d)
				}
			} else {
				if d.Name == name {
					listDyno(w, &d)
				}
			}
		}
	}
}

func listDyno(w io.Writer, d *heroku.Dyno) {
	listRec(w,
		d.Name,
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
