package empire

import (
	"io"

	"github.com/remind101/empire/empire/pkg/service"
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
	store   *store
	manager service.Manager
}

func (r *runnerService) Run(ctx context.Context, app *App, opts ProcessRunOpts) error {
	release, err := r.store.ReleasesFirst(ReleasesQuery{App: app})
	if err != nil {
		return err
	}

	a := &service.App{
		ID:   release.App.ID,
		Name: release.App.Name,
	}
	p := newServiceProcess(release, NewProcess("run", Command(opts.Command)))

	for k, v := range opts.Env {
		p.Env[k] = v
	}

	return r.manager.Run(ctx, a, p, opts.Input, opts.Output)
}
