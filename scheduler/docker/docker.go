// Package docker implements the Scheduler interface backed by the Docker API.
// This implementation is not recommended for production use, but can be used in
// development for testing.
package docker

import (
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/pkg/dockerutil"
	"github.com/remind101/empire/pkg/runner"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

type dockerClient interface {
	InspectContainer(string) (*docker.Container, error)
	ListContainers(docker.ListContainersOptions) ([]docker.APIContainers, error)
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
	runner *runner.Runner
	docker dockerClient
}

// NewScheduler returns a new Scheduler instance that uses the given client to
// interact with Docker.
func NewScheduler(client *dockerutil.Client) *Scheduler {
	return &Scheduler{
		runner: runner.NewRunner(client),
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
	return s.runner.Run(ctx, runner.RunOpts{
		Image:     p.Image,
		Command:   p.Command,
		Env:       scheduler.Env(app, p),
		Memory:    int64(p.MemoryLimit),
		CPUShares: int64(p.CPUShares),
		Labels:    labels,
		Input:     in,
		Output:    out,
	})
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
