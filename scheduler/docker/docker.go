// Package docker implements the Scheduler interface backed by the Docker API.
package docker

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"code.google.com/p/go-uuid/uuid"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/scheduler"
)

// The amount of time to wait for a container to stop before sending a SIGKILL.
const stopContainerTimeout = 10 // Seconds

// dockerClient defines the Docker client interface we use.
type dockerClient interface {
	InspectContainer(string) (*docker.Container, error)
	ListContainers(docker.ListContainersOptions) ([]docker.APIContainers, error)
	PullImage(context.Context, docker.PullImageOptions) error
	CreateContainer(context.Context, docker.CreateContainerOptions) (*docker.Container, error)
	RemoveContainer(context.Context, docker.RemoveContainerOptions) error
	StartContainer(context.Context, string, *docker.HostConfig) error
	StopContainer(context.Context, string, uint) error
	AttachToContainer(context.Context, docker.AttachToContainerOptions) error
}

const (
	// Label that determines whether the container is from a one-off run or
	// not. The value of this label will be `attached` or `detached`.
	runLabel = "run"

	// Label that determines what app the run relates to.
	appLabel = "empire.app.id"

	// Label that determines what the name of the process is.
	processLabel = "empire.app.process"
)

// Values for `runLabel`.
const (
	Attached = "attached"
	Detached = "detached"
)

// AttachedScheduler wraps a Scheduler to run attached processes using the Docker
// scheduler.
type AttachedScheduler struct {
	// If set, attached run instances will be merged in with instances
	// returned from the wrapped scheduler. This is currently an
	// experimental feature, since it requires that multiple Empire
	// processes interact with a single Docker daemon.
	ShowAttached bool

	scheduler.Scheduler
	dockerScheduler *Scheduler
}

// RunAttachedWithDocker wraps a Scheduler to run attached Run's using a Docker
// client.
func RunAttachedWithDocker(s scheduler.Scheduler, client *dockerutil.Client) *AttachedScheduler {
	return &AttachedScheduler{
		Scheduler:       s,
		dockerScheduler: NewScheduler(client),
	}
}

// Run runs attached processes using the docker scheduler, and detached
// processes using the wrapped scheduler.
func (s *AttachedScheduler) Run(ctx context.Context, app *scheduler.App, process *scheduler.Process, in io.Reader, out io.Writer) error {
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
func (s *AttachedScheduler) Instances(ctx context.Context, app string) ([]*scheduler.Instance, error) {
	if !s.ShowAttached {
		return s.Scheduler.Instances(ctx, app)
	}

	type instancesResult struct {
		instances []*scheduler.Instance
		err       error
	}

	ch := make(chan instancesResult, 1)
	go func() {
		attachedInstances, err := s.dockerScheduler.InstancesFromAttachedRuns(ctx, app)
		ch <- instancesResult{attachedInstances, err}
	}()

	instances, err := s.Scheduler.Instances(ctx, app)
	if err != nil {
		return instances, err
	}

	result := <-ch
	if err := result.err; err != nil {
		return instances, err
	}

	return append(instances, result.instances...), nil
}

// Stop checks if there's an attached run matching the given id, and stops that
// container if there is. Otherwise, it delegates to the wrapped Scheduler.
func (s *AttachedScheduler) Stop(ctx context.Context, maybeContainerID string) error {
	if !s.ShowAttached {
		return s.Scheduler.Stop(ctx, maybeContainerID)
	}

	err := s.dockerScheduler.Stop(ctx, maybeContainerID)

	// If there's no container with this ID, delegate to the wrapped
	// scheduler.
	if _, ok := err.(*docker.NoSuchContainer); ok {
		return s.Scheduler.Stop(ctx, maybeContainerID)
	}

	return err
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
	labels[runLabel] = Attached

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
	// Filter only docker containers that were started as an attached run.
	attached := fmt.Sprintf("%s=%s", runLabel, Attached)
	return s.instances(ctx, app, attached)
}

// instances returns docker container instances for this app, optionally
// filtered with labels.
func (s *Scheduler) instances(ctx context.Context, app string, labels ...string) ([]*scheduler.Instance, error) {
	var instances []*scheduler.Instance

	containers, err := s.docker.ListContainers(docker.ListContainersOptions{
		Filters: map[string][]string{
			"label": append([]string{
				fmt.Sprintf("%s=%s", appLabel, app),
			}, labels...),
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

// Stop stops the given container.
func (s *Scheduler) Stop(ctx context.Context, containerID string) error {
	container, err := s.docker.InspectContainer(containerID)
	if err != nil {
		return err
	}

	// Some extra protection around stopping containers. We don't want to
	// allow users to stop containers that may have been started outside of
	// Empire.
	if _, ok := container.Config.Labels[runLabel]; !ok {
		return &docker.NoSuchContainer{
			ID: containerID,
		}
	}

	if err := s.docker.StopContainer(ctx, containerID, stopContainerTimeout); err != nil {
		return err
	}

	return nil
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
