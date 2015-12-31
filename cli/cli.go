package cli

import (
	"fmt"
	"io"
	"strings"
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
func (c *Context) Args() []string           { return c.CLIContext.Args() }

// Empire mocks out the public interface for Empire.
type Empire interface {
	AppsFind(empire.AppsQuery) (*empire.App, error)
	Restart(context.Context, empire.RestartOpts) error
	Tasks(context.Context, *empire.App) ([]*empire.Task, error)
	Run(context.Context, empire.RunOpts) error
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
			Name:    "tasks",
			Aliases: []string{"ps"},
			Usage:   "Lists tasks for an application",
			Action:  wrap(c.Tasks),
			Flags: []cli.Flag{
				appFlag,
			},
		},
		{
			Name:   "restart",
			Usage:  "Restarts an application",
			Action: wrap(c.Restart),
			Flags: []cli.Flag{
				appFlag,
			},
		},
		{
			Name:   "run",
			Usage:  "Run a one off task",
			Action: wrap(c.RunTask),
			Flags: []cli.Flag{
				appFlag,
				cli.BoolFlag{
					Name:  "detached, d",
					Usage: "Run in detached mode instead of attached to terminal",
				},
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

// RunTask runs the `run` subcommand, which starts a one off task.
func (c *CLI) RunTask(ctx *Context, stdout io.Writer) error {
	app, err := c.findApp(ctx)
	if err != nil {
		return err
	}

	command := strings.Join(ctx.Args(), " ")
	if err := c.Empire.Run(ctx, empire.RunOpts{
		User:    empire.UserFromContext(ctx),
		App:     app,
		Command: command,
	}); err != nil {
		return err
	}

	fmt.Fprintf(stdout, "Ran `%s` on %s, detached\n", command, app.Name)
	return nil
}

func (c *CLI) findApp(ctx *Context) (*empire.App, error) {
	name := ctx.String("app")
	a, err := c.Empire.AppsFind(empire.AppsQuery{Name: &name})
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
