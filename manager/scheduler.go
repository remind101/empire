package manager

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/schema"
	"github.com/remind101/empire/images"
)

// Scheduler is an interface that represents something that can schedule Jobs
// onto the cluster.
type Scheduler interface {
	// Schedule schedules a job to run on the cluster.
	Schedule(*Job) error

	// ScheduleMulti schedules multiple jobs to run on the cluster.
	ScheduleMulti([]*Job) error
}

// NewScheduler is a factory method for generating a new Scheduler instance.
func NewScheduler(fleet string) (Scheduler, error) {
	if fleet == "" {
		return nil, nil
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

// ScheduleMulti implements Scheduler ScheduleMulti.
func (s *scheduler) ScheduleMulti(jobs []*Job) error {
	return nil
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

func (s *FleetScheduler) ScheduleMulti(jobs []*Job) error {
	for _, j := range jobs {
		if err := s.Schedule(j); err != nil {
			return err
		}
	}

	return nil
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
			Value:   fmt.Sprintf(`/usr/bin/docker run --name %s --rm -h %%H -P %s %s %s`, j.Name, img, env(j), j.Execute.Command),
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

func image(i images.Image) string {
	return fmt.Sprintf("quay.io/%s:%s", i.Repo, i.ID)
}

func env(j *Job) string {
	envs := []string{}
	for k, v := range j.Environment {
		envs = append(envs, fmt.Sprintf("-e %s=%s", k, v))
	}

	return strings.Join(envs, " ")
}
