package scheduler

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/schema"
)

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
func (s *FleetScheduler) Schedule(c *Container) error {
	u := s.buildUnit(c)
	return s.client.CreateUnit(u)
}

func (s *FleetScheduler) Unschedule(n ContainerName) error {
	return s.client.DestroyUnit(unitNameFromContainerName(n))
}

func (s *FleetScheduler) ContainerStates() ([]*ContainerState, error) {
	states, err := s.client.UnitStates()
	if err != nil {
		return nil, err
	}

	js := make([]*ContainerState, len(states))
	for i, st := range states {
		js[i] = &ContainerState{
			MachineID: st.MachineID,
			Name:      jobNameFromUnitName(st.Name),
			State:     st.SystemdActiveState,
		}
	}

	return js, nil
}

func unitNameFromContainerName(n ContainerName) string {
	return string(n) + ".service"
}

func jobNameFromUnitName(un string) ContainerName {
	return ContainerName(strings.TrimSuffix(un, ".service"))
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

func (s *FleetScheduler) buildUnit(c *Container) *schema.Unit {
	img := image(c.Image)
	opts := []*schema.UnitOption{
		{
			Section: "Unit",
			Name:    "Description",
			Value:   string(c.Name),
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
			Value:   fmt.Sprintf(`/bin/bash -c "/usr/bin/docker rm %s &> /dev/null; exit 0"`, c.Name),
		},
		{
			Section: "Service",
			Name:    "ExecStart",
			Value:   fmt.Sprintf(`/usr/bin/docker run --name %s %s --rm -h %%H -P %s %s`, c.Name, env(c), img, c.Command),
		},
		{
			Section: "Service",
			Name:    "ExecStop",
			Value:   fmt.Sprintf(`/usr/bin/docker stop %s`, c.Name),
		},
	}

	return &schema.Unit{
		Name:         string(c.Name) + ".service",
		DesiredState: "launched",
		Options:      opts,
	}
}

func image(i Image) string {
	return fmt.Sprintf("quay.io/%s:%s", i.Repo, i.ID)
}

func env(c *Container) string {
	envs := []string{}
	for k, v := range c.Environment {
		envs = append(envs, fmt.Sprintf("-e %s=%s", k, v))
	}

	return strings.Join(envs, " ")
}
