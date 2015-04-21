package empire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/remind101/empire/empire/pkg/container"
	"github.com/remind101/empire/empire/pkg/pod"
	"github.com/remind101/pkg/timex"
	"golang.org/x/net/context"
)

type ContainerRelay struct {
	Name      string `json:"name"`
	AttachURL string `json:"attach_url"`
	Command   string `json:"command"`
	State     string `json:"state"`
	Type      string
	Size      string
	CreatedAt time.Time `json:"created_at"`
}

// containerRelayer defines an interface for running a container
// remotely.
type containerRelayer interface {
	Relay(context.Context, *container.Container) (*ContainerRelay, error)
}

type fakeRelayer struct{}

func (f *fakeRelayer) Relay(ctx context.Context, c *container.Container) (*ContainerRelay, error) {
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

type relayer struct {
	API string // Location of the relay http api.
}

type postContainersForm struct {
	Image   string            `json:"image"`
	Command string            `json:"command"`
	Env     map[string]string `json:"env"`
	Attach  bool              `json:"attach"`
}

func (r *relayer) Relay(ctx context.Context, c *container.Container) (*ContainerRelay, error) {
	f := &postContainersForm{
		Image:   c.Image.String(),
		Command: c.Command,
		Env:     c.Env,
		Attach:  true,
	}

	b, err := json.Marshal(f)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/containers", r.API)
	resp, err := http.Post(url, "application/json", bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	cr := &ContainerRelay{}
	rb, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(rb, cr)
	if err != nil {
		return nil, err
	}

	return cr, nil
}

type runner struct {
	store   *store
	relayer containerRelayer
}

func (r *runner) Run(ctx context.Context, app *App, command string, opts ProcessesRunOpts) (*ContainerRelay, error) {
	c, err := r.newContainer(ctx, app, command, opts)
	if err != nil {
		return nil, err
	}

	return r.relayer.Relay(ctx, c)
}

func (r *runner) newContainer(ctx context.Context, app *App, command string, opts ProcessesRunOpts) (*container.Container, error) {
	release, err := r.store.ReleasesLast(app)
	if err != nil {
		return nil, err
	}

	if release == nil {
		return nil, &ValidationError{Err: fmt.Errorf("no releases for %s", app.Name)}
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
