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

// CLI represents a CLI interface to Empire.
type CLI struct {
	Empire
	io.Writer
}

func NewInternal(e *empire.Empire) *CLI {
	return New(e)
}

// New returns a new CLI instance.
func New(e Empire) *CLI {
	return &CLI{
		Empire: e,
	}
}

// appFlag is a common flag used in most commands to scope it to a specific
// application.
var appFlag = cli.StringFlag{
	Name:  "app, a",
	Usage: "The application",
	Value: "",
}

// Context wraps a cli.Context and context.Context as a single Context object.
type Context struct {
	*cli.Context
	ctx
}

// ctx aliases the context.Context interface.
type ctx context.Context

// Empire mocks out the public interface for Empire.
type Empire interface {
	Apps(empire.AppsQuery) ([]*empire.App, error)
	AppsFind(empire.AppsQuery) (*empire.App, error)
	Restart(context.Context, empire.RestartOpts) error
	Tasks(context.Context, *empire.App) ([]*empire.Task, error)
	Run(context.Context, empire.RunOpts) error
	Scale(context.Context, empire.ScaleOpts) (*empire.Process, error)
}

// Run runs the command.
func (c *CLI) Run(ctx context.Context, args []string) error {
	return run(ctx, c, args)
}

// Tasks runs the `tasks` subcommand, which lists the running and pending tasks
// for an application.
func (c *CLI) Tasks(ctx *Context) error {
	app, err := c.findApp(ctx)
	if err != nil {
		return err
	}

	tasks, err := c.Empire.Tasks(ctx, app)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(c, 1, 2, 2, ' ', 0)
	defer w.Flush()

	for _, task := range tasks {
		listRec(w, task.Name, task.Constraints.String(), task.State)
	}

	return nil
}

// Restart runs the `restart` subcommand, which restarts an application.
func (c *CLI) Restart(ctx *Context) error {
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

	fmt.Fprintf(c, "Restarted %s\n", app.Name)
	return nil
}

// RunTask runs the `run` subcommand, which starts a one off task.
func (c *CLI) RunTask(ctx *Context) error {
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

	fmt.Fprintf(c, "Ran `%s` on %s, detached\n", command, app.Name)
	return nil
}

// Apps runs the `apps` subcommand, which lists applications.
func (c *CLI) Apps(ctx *Context) error {
	all := empire.AppsQuery{}
	apps, err := c.Empire.Apps(all)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(c, 1, 2, 2, ' ', 0)
	defer w.Flush()

	for _, app := range apps {
		listRec(w, app.Name)
	}

	return nil
}

// Scale runs the `scale` subcommand, which scales processes.
func (c *CLI) Scale(ctx *Context) error {
	app, err := c.findApp(ctx)
	if err != nil {
		return err
	}

	var todo []empire.ScaleOpts
	types := make(map[string]bool)
	for _, arg := range ctx.Args() {
		pstype, qty, _, err := parseScaleArg(arg)
		if err != nil {
			return err
		}

		if _, exists := types[pstype]; exists {
			return fmt.Errorf("process type '%s' specified more than once", pstype)
		}
		types[pstype] = true

		todo = append(todo, empire.ScaleOpts{
			User:     empire.UserFromContext(ctx),
			App:      app,
			Process:  empire.ProcessType(pstype),
			Quantity: qty,
		})
	}

	for _, opts := range todo {
		if _, err := c.Empire.Scale(ctx, opts); err != nil {
			return err
		}
	}

	fmt.Fprintf(c, "Scaled %s", app.Name)
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

func newApp(w io.Writer) *cli.App {
	app := cli.NewApp()
	app.Name = "emp"
	app.HelpName = "emp"
	app.Usage = "CLI for Empire"
	app.Version = empire.Version
	app.Writer = w
	return app
}

// run runs a CLI command.
func run(ctx context.Context, c *CLI, args []string) (err error) {
	app := newApp(c)

	wrap := func(fn func(*Context) error) func(*cli.Context) {
		return func(clictx *cli.Context) {
			if subErr := fn(&Context{Context: clictx, ctx: ctx}); subErr != nil {
				err = subErr
			}
		}
	}

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
		{
			Name:   "apps",
			Usage:  "Lists applications",
			Action: wrap(c.Apps),
		},
		{
			Name:   "scale",
			Usage:  "Change process quantities and sizes",
			Action: wrap(c.Scale),
			Flags: []cli.Flag{
				appFlag,
			},
		},
	}

	err2 := app.Run(args)
	if err2 != nil {
		if err == nil {
			err = err2
		}
	}
	return
}
