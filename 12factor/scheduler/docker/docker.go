package docker

import (
	"github.com/fsouza/go-dockerclient"
	"github.com/remind101/empire/12factor"
)

// dockerClient represents the docker Client.
type dockerClient interface{}

// Scheduler is an implementation of the twelvefactor.Scheduler interface that
// talks to the Docker daemon API.
type Scheduler struct {
	docker dockerClient
}

// NewScheduler returns a new Scheduler instance backed by the docker client.
func NewScheduler(c *docker.Client) *Scheduler {
	return &Scheduler{
		docker: c,
	}
}

// NewSchedulerFromEnv returns a new Scheduler instance with a Docker client
// configured from the environment.
func NewSchedulerFromEnv() (*Scheduler, error) {
	c, err := docker.NewClientFromEnv()
	if err != nil {
		return nil, err
	}
	return NewScheduler(c), nil
}

// Run runs the application with Docker.
func (s *Scheduler) Run(manifest twelvefactor.Manifest) error {
	return nil
}
