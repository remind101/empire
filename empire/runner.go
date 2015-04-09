package empire

import (
	"time"

	"github.com/remind101/empire/empire/pkg/container"
	"github.com/remind101/empire/empire/pkg/pod"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type ContainerRelay struct {
	Name      string
	AttachURL string
	Command   string
	State     string
	Type      string
	Size      string
	CreatedAt time.Time
}

// ContainerRelayer defines an interface for running a container
// remotely.
type ContainerRelayer interface {
	Relay(*container.Container) (*ContainerRelay, error)
}

type fakeRelayer struct{}

func (f *fakeRelayer) Relay(c *container.Container) (*ContainerRelay, error) {
	return &ContainerRelay{
		Name:      "run.123",
		AttachURL: "fake://example.com:5000/abc",
		Command:   c.Command,
		State:     "starting",
		Type:      "run",
		Size:      "1X",
		CreatedAt: timex.Now(),
	}, nil
}

type runner struct {
	store   *store
	relayer ContainerRelayer
}

func (r *runner) Run(ctx context.Context, app *App, command string, opts ProcessesRunOpts) (*ContainerRelay, error) {
	c, err := r.newContainer(app, command, opts)
	if err != nil {
		return nil, err
	}

	return r.relayer.Relay(c)
}

func (r *runner) newContainer(app *App, command string, opts ProcessesRunOpts) (*container.Container, error) {
	release, err := r.store.ReleasesLast(app)
	if err != nil {
		return nil, err
	}

	config, err := r.store.ConfigsFind(release.ConfigID)
	if err != nil {
		return nil, err
	}

	slug, err := r.store.SlugsFind(release.SlugID)
	if err != nil {
		return nil, err
	}

	process := &Process{
		Type:     "run",
		Command:  Command(command),
		Quantity: 1,
	}

	vars := Vars{}
	for key, val := range opts.Env {
		vars[Variable(key)] = val
	}

	t := newTemplate(release, NewConfig(config, vars), slug, process)
	i := pod.NewInstance(t, 1)
	c := pod.NewContainer(i)
	return c, nil
}
