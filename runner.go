package empire

import (
	"io"

	"github.com/remind101/empire/scheduler"

	"golang.org/x/net/context"
)

type ProcessRunOpts struct {
	Command string

	// If provided, input will be read from this.
	Input io.Reader

	// If provided, output will be written to this.
	Output io.Writer

	// Extra environment variables to set.
	Env map[string]string
}

type runnerService struct {
	store     *store
	scheduler scheduler.Scheduler
}

func (r *runnerService) Run(ctx context.Context, app *App, opts ProcessRunOpts) error {
	release, err := r.store.ReleasesFirst(ReleasesQuery{App: app})
	if err != nil {
		return err
	}

	a := newServiceApp(release)
	p := newServiceProcess(release, NewProcess("run", Command(opts.Command)))

	for k, v := range opts.Env {
		p.Env[k] = v
	}

	return r.scheduler.Run(ctx, a, p, opts.Input, opts.Output)
}
