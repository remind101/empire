// package runner provides a simple interface for running docker containers.
package runner

import (
	"fmt"
	"io"
	"strings"

	"code.google.com/p/go-uuid/uuid"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/empire/pkg/dockerutil"
	"github.com/remind101/empire/empire/pkg/image"
	"golang.org/x/net/context"
)

// DefaultStopTimeout is the number of seconds to wait when stopping a
// container.
const DefaultStopTimeout = 10

// RunOpts is used when running.
type RunOpts struct {
	// Image is the image to run.
	Image image.Image

	// Command is the command to run.
	Command string

	// Environment variables to set.
	Env map[string]string

	// Streams fo Stdout, Stderr and Stdin.
	Input  io.Reader
	Output io.Writer
}

// Runner is a service for running containers.
type Runner struct {
	client *dockerutil.Client
}

// NewRunner returns a new Runner instance using the docker.Client as the docker
// client.
func NewRunner(client *dockerutil.Client) *Runner {
	return &Runner{client: client}
}

func (r *Runner) Run(ctx context.Context, opts RunOpts) error {
	if err := r.pull(opts.Image, opts.Output); err != nil {
		return fmt.Errorf("runner: pull: %v", err)
	}

	c, err := r.create(opts)
	if err != nil {
		return fmt.Errorf("runner: create container: %v", err)
	}
	defer r.remove(c.ID)

	if err := r.start(c.ID); err != nil {
		return fmt.Errorf("runner: start containeer: %v", err)
	}

	if err := r.attach(c.ID, opts.Input, opts.Output); err != nil {
		return fmt.Errorf("runner: attach: %v", err)
	}
	defer tryClose(opts.Output)

	if err := r.wait(c.ID); err != nil {
		return fmt.Errorf("runner: wait: %v", err)
	}

	if err := r.stop(c.ID); err != nil {
		if _, ok := err.(*docker.ContainerNotRunning); ok {
			return nil
		}

		return fmt.Errorf("runner: stop: %v", err)
	}

	return nil
}

func (r *Runner) pull(img image.Image, out io.Writer) error {
	return r.client.PullImage(docker.PullImageOptions{
		Registry:     img.Registry,
		Repository:   img.Repository,
		Tag:          img.Tag,
		OutputStream: out,
	})
}

func (r *Runner) create(opts RunOpts) (*docker.Container, error) {
	return r.client.CreateContainer(docker.CreateContainerOptions{
		Name: uuid.New(),
		Config: &docker.Config{
			Tty:          true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			Image:        opts.Image.String(),
			Cmd:          strings.Split(opts.Command, " "),
			Env:          envKeys(opts.Env),
		},
		HostConfig: &docker.HostConfig{},
	})
}

func (r *Runner) start(id string) error {
	return r.client.StartContainer(id, nil)
}

func (r *Runner) attach(id string, in io.Reader, out io.Writer) error {
	return r.client.AttachToContainer(docker.AttachToContainerOptions{
		Container:    id,
		InputStream:  in,
		OutputStream: out,
		ErrorStream:  out,
		Logs:         true,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	})
}

func (r *Runner) wait(id string) error {
	_, err := r.client.WaitContainer(id)
	return err
}

func (r *Runner) stop(id string) error {
	return r.client.StopContainer(id, DefaultStopTimeout)
}

func (r *Runner) remove(id string) error {
	return r.client.RemoveContainer(docker.RemoveContainerOptions{
		ID:            id,
		RemoveVolumes: true,
		Force:         true,
	})
}

func envKeys(env map[string]string) []string {
	var s []string

	for k, v := range env {
		s = append(s, fmt.Sprintf("%s=%s", k, v))
	}

	return s
}

func tryClose(w io.Writer) error {
	if w, ok := w.(io.Closer); ok {
		return w.Close()
	}

	return nil
}
