package empire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/remind101/empire/empire/pkg/runner"
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

type postContainersForm struct {
	User    string            `json:"user"`
	Image   string            `json:"image"`
	Command string            `json:"command"`
	Env     map[string]string `json:"env"`
	Attach  bool              `json:"attach"`
}

// containerRelayer defines an interface for running a container
// remotely.
type containerRelayer interface {
	Relay(context.Context, *postContainersForm) (*ContainerRelay, error)
}

type fakeRelayer struct{}

func (f *fakeRelayer) Relay(ctx context.Context, c *postContainersForm) (*ContainerRelay, error) {
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

func (r *relayer) Relay(ctx context.Context, f *postContainersForm) (*ContainerRelay, error) {
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

type fakeRunner struct{}

func (r *fakeRunner) Run(_ context.Context, opts runner.RunOpts) error {
	return nil
}

type runnerService struct {
	store  *store
	runner interface {
		Run(context.Context, runner.RunOpts) error
	}
}

func (r *runnerService) Run(ctx context.Context, app *App, command string, in io.Reader, out io.Writer) error {
	opts, err := r.newRunOpts(ctx, app, command)
	if err != nil {
		return err
	}

	opts.Input = in
	opts.Output = out

	return r.runner.Run(ctx, opts)
}

func (r *runnerService) newRunOpts(ctx context.Context, app *App, command string) (runner.RunOpts, error) {
	release, err := r.store.ReleasesFirst(ReleasesQuery{App: app})
	if err != nil {
		return runner.RunOpts{}, err
	}

	env := environment(release.Config.Vars)

	return runner.RunOpts{
		Image:   release.Slug.Image,
		Command: command,
		Env:     env,
	}, nil
}
