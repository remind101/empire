package empire

import (
	"io"

	"golang.org/x/net/context"
)

// RunRecorder is a function that returns an io.Writer that will be written to
// to record Stdout and Stdin of interactive runs.
type RunRecorder func() (io.Writer, error)

type runnerService struct {
	*Empire
}

func (r *runnerService) Run(ctx context.Context, opts RunOpts) error {
	release, err := releasesFind(r.db, ReleasesQuery{App: opts.App})
	if err != nil {
		return err
	}

	procName := opts.Command[0]
	proc := Process{
		Quantity: 1,
	}

	if cmd, ok := release.Formation[procName]; ok {
		proc.Command = append(cmd.Command, opts.Command[1:]...)
	} else {
		if r.AllowedCommands == AllowCommandProcfile {
			return commandNotInFormation(Command{procName}, release.Formation)
		}

		// This is an unnamed command, fallback to a generic proc name.
		procName = "run"
		proc.Command = opts.Command
	}

	// Set the size of the process.
	constraints := DefaultConstraints
	if opts.Constraints != nil {
		constraints = *opts.Constraints
	}
	proc.SetConstraints(constraints)

	a, err := newSchedulerApp(release)
	if err != nil {
		return err
	}
	p, err := newSchedulerProcess(release, procName, proc)
	if err != nil {
		return err
	}
	p.Labels["empire.user"] = opts.User.Name

	// Add additional environment variables to the process.
	for k, v := range opts.Env {
		p.Env[k] = v
	}

	return r.Scheduler.Run(ctx, a, p, opts.Input, opts.Output)
}
