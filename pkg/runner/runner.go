// package runner provides a simple interface for running docker containers.
package runner

import (
	"bytes"
	"fmt"
	"io"

	"code.google.com/p/go-uuid/uuid"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
	"golang.org/x/net/context"
)

// DefaultStopTimeout is the number of seconds to wait when stopping a
// container.
const DefaultStopTimeout = 10

// RunOpts is used when running.
type RunOpts struct {
	// Image is the image to run.
	Image string

	// Command is the command to run.
	Command []string

	// Environment variables to set.
	Env map[string]string

	// Labels to set
	Labels map[string]string

	// Memory/CPUShares.
	Memory    int64
	CPUShares int64

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
	if err := r.pull(ctx, opts.Image, replaceNL(opts.Output)); err != nil {
		return fmt.Errorf("runner: pull: %v", err)
	}

	c, err := r.create(ctx, opts)
	if err != nil {
		return fmt.Errorf("runner: create container: %v", err)
	}
	defer r.remove(ctx, c.ID)

	if err := r.start(ctx, c.ID); err != nil {
		return fmt.Errorf("runner: start container: %v", err)
	}
	defer tryClose(opts.Output)

	if err := r.attach(ctx, c.ID, opts.Input, opts.Output); err != nil {
		return fmt.Errorf("runner: attach: %v", err)
	}

	return nil
}

func (r *Runner) pull(ctx context.Context, img string, out io.Writer) error {
	ref, err := dockerutil.ParseReference(img)
	if err != nil {
		return err
	}

	return r.client.PullImage(ctx, docker.PullImageOptions{
		Repository:   ref.Name(),
		Tag:          ref.Tag(),
		OutputStream: out,
	})
}

func (r *Runner) create(ctx context.Context, opts RunOpts) (*docker.Container, error) {
	return r.client.CreateContainer(ctx, docker.CreateContainerOptions{
		Name: uuid.New(),
		Config: &docker.Config{
			Tty:          true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			Memory:       opts.Memory,
			CPUShares:    opts.CPUShares,
			Image:        opts.Image,
			Cmd:          opts.Command,
			Env:          envKeys(opts.Env),
			Labels:       opts.Labels,
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{
				Type: "json-file",
			},
		},
	})
}

func (r *Runner) start(ctx context.Context, id string) error {
	return r.client.StartContainer(ctx, id, nil)
}

func (r *Runner) attach(ctx context.Context, id string, in io.Reader, out io.Writer) error {
	return r.client.AttachToContainer(ctx, docker.AttachToContainerOptions{
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

func (r *Runner) remove(ctx context.Context, id string) error {
	return r.client.RemoveContainer(ctx, docker.RemoveContainerOptions{
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

// replaceNL returns an io.Writer that will replace "\n" with "\r\n" in the
// stream.
var replaceNL = func(w io.Writer) io.Writer {
	o, n := []byte("\n"), []byte("\r\n")
	return writerFunc(func(p []byte) (int, error) {
		return w.Write(bytes.Replace(p, o, n, -1))
	})
}

// writerFunc is a function that implements io.Writer.
type writerFunc func([]byte) (int, error)

func (f writerFunc) Write(p []byte) (int, error) {
	return f(p)
}
