// package manager is responsible for scheduling releases onto the cluster.
package manager

import (
	"fmt"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/slugs"
)

// Name represents the (unique) name of a job. The convention is <app>.<type>.<instance>:
//
//	589611bf-4e6c-4fc2-a77a-d61dca412289.web.1
type Name string

// NewName returns a new Name with the proper format.
func NewName(id apps.ID, pt formations.ProcessType, i int) Name {
	return Name(fmt.Sprintf("%s.%s.%d", id, pt, i))
}

// Job is a job that can be scheduled.
type Job struct {
	// The unique name of the job.
	Name Name

	// A map of environment variables to set.
	Environment map[string]string

	// The command to run.
	Command string
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
	Job   *Job
	State State
}

// Scheduler is an interface that represents something that can schedule Jobs
// onto the cluster.
type Scheduler interface {
	// Schedule schedules a job to run on the cluster.
	Schedule(*Job) error

	// TODO Jobs returns all of the jobs currently scheduled onto the
	// cluster and their state..
	// JobStates() ([]*JobState, error)

	// TODO Depending on the scheduler, we'd probably need to unschedule old
	// jobs.
	// Unschedule(Name) error
}

// scheduler is a fake implementation of the Scheduler interface.
type scheduler struct{}

func newScheduler() *scheduler {
	return &scheduler{}
}

func (s *scheduler) Schedule(j *Job) error {
	return nil
}

// Service provides a layer of convenience over a Scheduler.
type Service struct {
	Scheduler
}

// NewService returns a new Service instance.
func NewService(s Scheduler) *Service {
	if s == nil {
		s = newScheduler()
	}

	return &Service{
		Scheduler: s,
	}
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (s *Service) ScheduleRelease(release *releases.Release) error {
	return s.scheduleApp(
		release.App,
		release.Config,
		release.Slug,
		release.Formations,
	)
}

func (s *Service) scheduleApp(app *apps.App, config *configs.Config, slug *slugs.Slug, formation formations.Formations) error {
	var jobs []*Job

	// Build jobs for each process type
	for _, f := range formation {
		// We don't want to create jobs for process types that weren't
		// defined in the Procfile.
		if _, found := slug.ProcessTypes[f.ProcessType]; !found {
			continue
		}

		command := string(slug.ProcessTypes[f.ProcessType])
		env := environment(config.Vars)

		// Build a Job for each instance of the process.
		for i := 1; i <= f.Count; i++ {
			j := &Job{
				Name:        NewName(app.ID, f.ProcessType, i),
				Environment: env,
				Command:     command,
			}

			jobs = append(jobs, j)
		}
	}

	// Schedule all of the jobs.
	for _, j := range jobs {
		if err := s.Scheduler.Schedule(j); err != nil {
			return err
		}
	}

	return nil
}

// environment coerces a configs.Vars into a map[string]string.
func environment(vars configs.Vars) map[string]string {
	env := make(map[string]string)

	for k, v := range vars {
		env[string(k)] = string(v)
	}

	return env
}
