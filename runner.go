package empire

import "golang.org/x/net/context"

type runnerService struct {
	*Empire
}

func (r *runnerService) Run(ctx context.Context, opts RunOpts) error {
	release, err := r.store.ReleasesFind(ReleasesQuery{App: opts.App})
	if err != nil {
		return err
	}

	a := newApp(release)
	p := newProcess(release, NewProcess("run", Command(opts.Command)))
	p.Stdout = opts.Output
	p.Stdin = opts.Input

	for k, v := range opts.Env {
		p.Env[k] = v
	}

	return r.Scheduler.Run(a, p)
}
