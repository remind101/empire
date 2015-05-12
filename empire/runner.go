package empire

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

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

type runner struct {
	store   *store
	relayer containerRelayer
}

func (r *runner) Run(ctx context.Context, app *App, command string, opts ProcessesRunOpts) (*ContainerRelay, error) {
	c, err := r.newContainerForm(ctx, app, command, opts)
	if err != nil {
		return nil, err
	}

	return r.relayer.Relay(ctx, c)
}

func (r *runner) newContainerForm(ctx context.Context, app *App, command string, opts ProcessesRunOpts) (*postContainersForm, error) {
	release, err := r.store.ReleasesLast(app)
	if err != nil {
		return nil, err
	}

	// Merge env vars with App env vars.
	vars := Vars{}
	for key, val := range opts.Env {
		vars[Variable(key)] = val
	}
	env := environment(NewConfig(release.Config, vars).Vars)

	username := ""
	if user, ok := UserFromContext(ctx); ok {
		username = user.Name
	}

	c := &postContainersForm{
		User:    username,
		Image:   release.Slug.Image.String(),
		Command: command,
		Env:     env,
		Attach:  true,
	}

	return c, nil
}
