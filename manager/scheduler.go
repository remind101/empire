package manager

import (
	"net/http"
	"net/url"

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
	u := &schema.Unit{
		Name:         string(j.Name) + ".service",
		DesiredState: "launched",
		Options: []*schema.UnitOption{
			{
				Section: "Service",
				Name:    "ExecStart",
				Value:   dockerRun(j.Name, j.Execute.Image),
			},
		},
	}

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

func dockerRun(name Name, image images.Image) string {
	return "/usr/bin/docker run --name " + string(name) + " --rm -h %H -P quay.io/" + string(image.Repo) + ":" + string(image.ID)
}
