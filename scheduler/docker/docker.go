// Package docker implements the Scheduler interface backed by the Docker API.
package docker

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"strings"

	"code.google.com/p/go-uuid/uuid"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

type dockerClient interface {
	InspectContainer(string) (*docker.Container, error)
	ListContainers(docker.ListContainersOptions) ([]docker.APIContainers, error)
	PullImage(context.Context, docker.PullImageOptions) error
	CreateContainer(context.Context, docker.CreateContainerOptions) (*docker.Container, error)
	RemoveContainer(context.Context, docker.RemoveContainerOptions) error
	StartContainer(context.Context, string, *docker.HostConfig) error
	AttachToContainer(context.Context, docker.AttachToContainerOptions) error
}

const (
	// Label that determines whether the container is from an attached run
	// or not. The value of this label will be the app id.
	attachedRunLabel = "attached-run"

	// Label that determines what app the run relates to.
	appLabel = "empire.app.id"

	// Label that determines what the name of the process is.
	processLabel = "empire.app.process"
)

// attachedScheduler wraps a Scheduler to run attached processes using the Docker
// scheduler.
type attachedScheduler struct {
	scheduler.Scheduler
	dockerScheduler *Scheduler
}

// RunAttachedWithDocker wraps a Scheduler to run attached Run's using a Docker
// client.
func RunAttachedWithDocker(s scheduler.Scheduler, client *dockerutil.Client) scheduler.Scheduler {
	return &attachedScheduler{
		Scheduler:       s,
		dockerScheduler: NewScheduler(client),
	}
}

// Run runs attached processes using the docker scheduler, and detached
// processes using the wrapped scheduler.
func (s *attachedScheduler) Run(ctx context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
	// Attached means stdout, stdin is attached.
	attached := out != nil || in != nil

	if attached {
		return s.dockerScheduler.Run(ctx, app, process, in, out)
	} else {
		return s.Scheduler.Run(ctx, app, process, in, out)
	}
}

// Instances returns a combination of instances from the wrapped scheduler, as
// well as instances from attached runs.
func (s *attachedScheduler) Instances(ctx context.Context, app string) ([]*scheduler.Instance, error) {
	instances, err := s.Scheduler.Instances(ctx, app)
	if err != nil {
		return instances, err
	}

	attachedInstances, err := s.dockerScheduler.InstancesFromAttachedRuns(ctx, app)
	if err != nil {
		return instances, err
	}

	return append(instances, attachedInstances...), nil
}

// Scheduler provides an implementation of the scheduler.Scheduler interface
// backed by Docker.
type Scheduler struct {
	docker dockerClient
}

// NewScheduler returns a new Scheduler instance that uses the given client to
// interact with Docker.
func NewScheduler(client *dockerutil.Client) *Scheduler {
	return &Scheduler{
		docker: client,
	}
}

func (s *Scheduler) Run(ctx context.Context, app *scheduler.App, p *scheduler.Process, in io.Reader, out io.Writer) error {
	attached := out != nil || in != nil

	if !attached {
		return errors.New("cannot run detached processes with Docker scheduler")
	}

	labels := scheduler.Labels(app, p)
	labels[attachedRunLabel] = "true"

	if err := s.docker.PullImage(ctx, docker.PullImageOptions{
		Registry:     p.Image.Registry,
		Repository:   p.Image.Repository,
		Tag:          p.Image.Tag,
		OutputStream: replaceNL(out),
	}); err != nil {
		return fmt.Errorf("error pulling image: %v", err)
	}

	container, err := s.docker.CreateContainer(ctx, docker.CreateContainerOptions{
		Name: uuid.New(),
		Config: &docker.Config{
			Tty:          true,
			AttachStdin:  true,
			AttachStdout: true,
			AttachStderr: true,
			OpenStdin:    true,
			Memory:       int64(p.MemoryLimit),
			CPUShares:    int64(p.CPUShares),
			Image:        p.Image.String(),
			Cmd:          p.Command,
			Env:          envKeys(scheduler.Env(app, p)),
			Labels:       labels,
		},
		HostConfig: &docker.HostConfig{
			LogConfig: docker.LogConfig{
				Type: "json-file",
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error creating container: %v", err)
	}
	defer s.docker.RemoveContainer(ctx, docker.RemoveContainerOptions{
		ID:            container.ID,
		RemoveVolumes: true,
		Force:         true,
	})

	if err := s.docker.StartContainer(ctx, container.ID, nil); err != nil {
		return fmt.Errorf("error starting container: %v", err)
	}
	defer tryClose(out)

	if err := s.docker.AttachToContainer(ctx, docker.AttachToContainerOptions{
		Container:    container.ID,
		InputStream:  in,
		OutputStream: out,
		ErrorStream:  out,
		Logs:         true,
		Stream:       true,
		Stdin:        true,
		Stdout:       true,
		Stderr:       true,
		RawTerminal:  true,
	}); err != nil {
		return fmt.Errorf("error attaching to container: %v", err)
	}

	return nil
}

func (s *Scheduler) Instances(ctx context.Context, app string) ([]*scheduler.Instance, error) {
	return s.InstancesFromAttachedRuns(ctx, app)
}

// InstancesFromAttachedRuns returns Instances that were started from attached
// runs.
func (s *Scheduler) InstancesFromAttachedRuns(ctx context.Context, app string) ([]*scheduler.Instance, error) {
	var instances []*scheduler.Instance

	containers, err := s.docker.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": []string{
				fmt.Sprintf("%s=true", attachedRunLabel),
				fmt.Sprintf("%s=%s", appLabel, app),
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("error listing containers from attached runs: %v", err)
	}

	for _, apiContainer := range containers {
		container, err := s.docker.InspectContainer(apiContainer.ID)
		if err != nil {
			return instances, fmt.Errorf("error inspecting container %s: %v", apiContainer.ID, err)
		}

		state := strings.ToUpper(container.State.StateString())

		instances = append(instances, &scheduler.Instance{
			ID:        container.ID[0:12],
			State:     state,
			UpdatedAt: container.State.StartedAt,
			Process: &scheduler.Process{
				Type:        container.Config.Labels[processLabel],
				Command:     container.Config.Cmd,
				Env:         parseEnv(container.Config.Env),
				MemoryLimit: uint(container.HostConfig.Memory),
				CPUShares:   uint(container.HostConfig.CPUShares),
			},
		})
	}

	return instances, nil
}

func parseEnv(env []string) map[string]string {
	m := make(map[string]string)
	for _, e := range env {
		parts := strings.SplitN(e, "=", 2)
		m[parts[0]] = parts[1]
	}
	return m
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
