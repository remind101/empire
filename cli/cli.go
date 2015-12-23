package cli

import (
	"fmt"
	"io"
	"text/tabwriter"

	"golang.org/x/net/context"

	"github.com/codegangsta/cli"
	"github.com/remind101/empire"
)

// appFlag is a common flag used in most commands to scope it to a specific
// application.
var appFlag = cli.StringFlag{
	Name:  "app, a",
	Usage: "The application",
	Value: "",
}

type Context struct {
	context.Context
	CLIContext *cli.Context
}

func (c *Context) String(key string) string { return c.CLIContext.String(key) }

// Empire mocks out the public interface for Empire.
type Empire interface {
	AppsFirst(empire.AppsQuery) (*empire.App, error)
	Restart(context.Context, empire.RestartOpts) error
	Tasks(context.Context, *empire.App) ([]*empire.Task, error)
}

// CLI represents a CLI interface to Empire.
type CLI struct {
	Empire

	// NewApp generates a new cli.App when Run is called.
	NewApp func() *cli.App
}

// NewApp generates a new cli.App for the Empire CLI.
func NewApp() *cli.App {
	app := cli.NewApp()
	app.Usage = "CLI for Empire"
	return app
}

// NewCLI returns a new CLI instance.
func NewCLI(e Empire) *CLI {
	return &CLI{
		Empire: e,
		NewApp: NewApp,
	}
}

// Run runs the command.
func (c *CLI) Run(ctx context.Context, stdout io.Writer, args []string) (err error) {
	app := c.NewApp()

	wrap := func(fn func(*Context, io.Writer) error) func(*cli.Context) {
		return func(clictx *cli.Context) {
			if subErr := fn(&Context{Context: ctx, CLIContext: clictx}, stdout); subErr != nil {
				err = subErr
			}
		}
	}

	app.Writer = stdout
	app.Commands = []cli.Command{
		{
			Name:   "tasks",
			Usage:  "list tasks for an application",
			Action: wrap(c.Tasks),
			Flags: []cli.Flag{
				appFlag,
			},
		},
		{
			Name:   "restart",
			Usage:  "restart an application",
			Action: wrap(c.Restart),
			Flags: []cli.Flag{
				appFlag,
			},
		},
	}

	app.Run(args)
	return
}

// Tasks runs the `tasks` subcommand, which lists the running and pending tasks
// for an application.
func (c *CLI) Tasks(ctx *Context, stdout io.Writer) error {
	app, err := c.findApp(ctx)
	if err != nil {
		return err
	}

	tasks, err := c.Empire.Tasks(ctx, app)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(stdout, 1, 2, 2, ' ', 0)
	defer w.Flush()

	for _, task := range tasks {
		listRec(w, task.Name, task.Constraints.String(), task.State)
	}

	return nil
}

// Restart runs the `restart` subcommand, which restarts an application.
func (c *CLI) Restart(ctx *Context, stdout io.Writer) error {
	app, err := c.findApp(ctx)
	if err != nil {
		return err
	}

	if err := c.Empire.Restart(ctx, empire.RestartOpts{
		User: empire.UserFromContext(ctx),
		App:  app,
	}); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Restarted %s\n", app.Name)
	return nil
}

func (c *CLI) findApp(ctx *Context) (*empire.App, error) {
	name := ctx.String("app")
	a, err := c.Empire.AppsFirst(empire.AppsQuery{Name: &name})
	return a, err
}

func listRec(w io.Writer, a ...interface{}) {
	for i, x := range a {
		fmt.Fprint(w, x)
		if i+1 < len(a) {
			w.Write([]byte{'\t'})
		} else {
			w.Write([]byte{'\n'})
		}
	}
}
