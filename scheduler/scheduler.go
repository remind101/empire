package scheduler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/schema"
)

// JobName represents the (unique) name of a job. The convention is <app>.<type>.<instance>:
//
//	my-sweet-app.v1.web.1
type JobName string

// Image represents a container image, which is tied to a repository.
type Image struct {
	Repo string
	ID   string
}

// Execute represents a command to execute inside and image.
type Execute struct {
	Command string
	Image   Image
}

// Job is a job that can be scheduled.
type Job struct {
	// The unique name of the job.
	Name JobName

	// A map of environment variables to set.
	Environment map[string]string

	// The command to run.
	Execute Execute
}

// State represents the state of a job.
type State int

// Various states that a job can be in.
const (
	StatePending State = iota
	StateRunning
	StateFailed
)

// JobState represents the status of a job.
type JobState struct {
	MachineID string
	Name      JobName
	State     string // TODO use State type
}

// Scheduler is an interface that represents something that can schedule Jobs
// onto the cluster.
type Scheduler interface {
	// Schedule schedules a job to run on the cluster.
	Schedule(*Job) error

	// Unschedule unschedules a job from the cluster by its name.
	Unschedule(JobName) error

	// List JobStates
	JobStates() ([]*JobState, error)
}

// NewScheduler is a factory method for generating a new Scheduler instance.
func NewScheduler(fleet string) (Scheduler, error) {
	if fleet == "" {
		return newScheduler(), nil
	}

	return newFleetScheduler(fleet)
}

// scheduler is a fake implementation of the Scheduler interface.
type scheduler struct{}

func newScheduler() *scheduler {
	return &scheduler{}
}

// Schedule implements Scheduler Schedule.
func (s *scheduler) Schedule(j *Job) error {
	return nil
}

// Unschedule implements Scheduler Unschedule.
func (s *scheduler) Unschedule(n JobName) error {
	return nil
}

func (s *scheduler) JobStates() ([]*JobState, error) {
	return nil, nil
}

// FleetScheduler is an implementation of the Scheduler interface that schedules
// jobs onto a coreos cluster via the fleet api.
type FleetScheduler struct {
	client client.API
}

// newFleetScheduler returns a new FleetScheduler with a configured fleet api
// client.
func newFleetScheduler(fleet string) (*FleetScheduler, error) {
	u, err := url.Parse(fleet)
	if err != nil {
		return nil, err
	}

	c, err := client.NewHTTPClient(
		http.DefaultClient,
		*u,
	)
	if err != nil {
		panic(err)
	}

	return &FleetScheduler{
		client: c,
	}, nil
}

// Schedule implements Scheduler Schedule and builds an appropritate systemd
// unit definition to run the container.
func (s *FleetScheduler) Schedule(j *Job) error {
	u := s.buildUnit(j)
	return s.client.CreateUnit(u)
}

func (s *FleetScheduler) Unschedule(n JobName) error {
	return s.client.DestroyUnit(string(n) + ".service")
}

func (s *FleetScheduler) JobStates() ([]*JobState, error) {
	states, err := s.client.UnitStates()
	if err != nil {
		return nil, err
	}

	js := make([]*JobState, len(states))
	for i, st := range states {
		js[i] = &JobState{
			MachineID: st.MachineID,
			Name:      JobName(st.Name),
			State:     st.SystemdActiveState,
		}
	}

	return js, nil
}

// buildUnit builds a Unit file that looks like this:
//
// [Unit]
// Description=app.v1.web.1
// After=discovery.service
//
// [Service]
// TimeoutStartSec=30m
//
// ExecStartPre=/bin/bash -c "/usr/bin/docker inspect remind101/app &> /dev/null || /usr/bin/docker pull remind101/app"
// ExecStartPre=/bin/bash -c "/usr/bin/docker rm app.v1.web.1 &> /dev/null; exit 0"
// ExecStart=/usr/bin/docker run --name app.v1.web.1 --rm -h %H remind101/app
// ExecStop=/usr/bin/docker stop app.v1.web.1

func (s *FleetScheduler) buildUnit(j *Job) *schema.Unit {
	img := image(j.Execute.Image)
	opts := []*schema.UnitOption{
		{
			Section: "Unit",
			Name:    "Description",
			Value:   string(j.Name),
		},
		{
			Section: "Unit",
			Name:    "After",
			Value:   "discovery.service",
		},
		{
			Section: "Service",
			Name:    "TimeoutStartSec",
			Value:   "30m",
		},
		{
			Section: "Service",
			Name:    "Restart",
			Value:   "on-failure",
		},
		{
			Section: "Service",
			Name:    "ExecStartPre",
			Value:   fmt.Sprintf(`/bin/bash -c "/usr/bin/docker inspect %s &> /dev/null || /usr/bin/docker pull %s"`, img, img),
		},
		{
			Section: "Service",
			Name:    "ExecStartPre",
			Value:   fmt.Sprintf(`/bin/bash -c "/usr/bin/docker rm %s &> /dev/null; exit 0"`, j.Name),
		},
		{
			Section: "Service",
			Name:    "ExecStart",
			Value:   fmt.Sprintf(`/usr/bin/docker run --name %s %s --rm -h %%H -P %s %s`, j.Name, env(j), img, j.Execute.Command),
		},
		{
			Section: "Service",
			Name:    "ExecStop",
			Value:   fmt.Sprintf(`/usr/bin/docker stop %s`, j.Name),
		},
	}

	return &schema.Unit{
		Name:         string(j.Name) + ".service",
		DesiredState: "launched",
		Options:      opts,
	}
}

func image(i Image) string {
	return fmt.Sprintf("quay.io/%s:%s", i.Repo, i.ID)
}

func env(j *Job) string {
	envs := []string{}
	for k, v := range j.Environment {
		envs = append(envs, fmt.Sprintf("-e %s=%s", k, v))
	}

	return strings.Join(envs, " ")
}
