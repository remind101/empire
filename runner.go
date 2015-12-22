package empire

import "golang.org/x/net/context"

type runnerService struct {
	*Empire
}

func (r *runnerService) Run(ctx context.Context, opts RunOpts) error {
	release, err := r.store.ReleasesFirst(ReleasesQuery{App: opts.App})
	if err != nil {
		return err
	}

	a := newServiceApp(release)
	p := newServiceProcess(release, NewProcess("run", Command(opts.Command)))

	for k, v := range opts.Env {
		p.Env[k] = v
	}

	return r.Scheduler.Run(ctx, a, p, opts.Input, opts.Output)
}
