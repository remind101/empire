package empire

import (
	"io"

	"github.com/remind101/empire/empire/pkg/service"
	"golang.org/x/net/context"
)

type runnerService struct {
	store   *store
	manager service.Manager
}

func (r *runnerService) Run(ctx context.Context, app *App, command string, in io.Reader, out io.Writer) error {
	release, err := r.store.ReleasesFirst(ReleasesQuery{App: app})
	if err != nil {
		return err
	}

	a := &service.App{
		ID:   release.App.ID,
		Name: release.App.Name,
	}
	p := newServiceProcess(release, NewProcess("run", Command(command)))
	return r.manager.Run(ctx, a, p, in, out)
}
