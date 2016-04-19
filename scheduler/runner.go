package scheduler

import (
	"io"

	"github.com/remind101/empire/pkg/runner"
	"golang.org/x/net/context"
)

// AttachedRunner wraps a Manager to run attached processes using docker
// directly to get access to stdin and stdout.
type AttachedRunner struct {
	Scheduler
	Runner *runner.Runner
}

func (m *AttachedRunner) Run(ctx context.Context, app *App, p *Process, in io.Reader, out io.Writer) error {
	// If an output stream is provided, run using the docker runner.
	if out != nil {
		return m.Runner.Run(ctx, runner.RunOpts{
			Image:   app.Image,
			Command: p.Command,
			Env:     ProcessEnv(app, p),
			Input:   in,
			Output:  out,
		})
	}

	return m.Scheduler.Run(ctx, app, p, in, out)
}
