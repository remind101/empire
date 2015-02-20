// package manager is responsible for scheduling releases onto the cluster.
package manager

import (
	"fmt"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/processes"
	"github.com/remind101/empire/releases"
)

// Name represents the (unique) name of a job. The convention is <app>.<type>.<instance>:
//
//	my-sweet-app.v1.web.1
type Name string

// NewName returns a new Name with the proper format.
func NewName(id apps.Name, v releases.Version, pt processes.Type, i int) Name {
	return Name(fmt.Sprintf("%s.%s.%s.%d", id, v, pt, i))
}

// Execute represents a command to execute inside and image.
type Execute struct {
	Command string
	Image   images.Image
}

// Job is a job that can be scheduled.
type Job struct {
	// The unique name of the job.
	Name Name

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
	Job   *Job
	State State
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
	jobs := buildJobs(
		release.App.Name,
		release.Version,
		*release.Slug.Image,
		release.Config.Vars,
		release.Formation,
	)

	return s.Scheduler.ScheduleMulti(jobs)
}

func buildJobs(name apps.Name, version releases.Version, image images.Image, vars configs.Vars, formation []*formations.CommandFormation) []*Job {
	var jobs []*Job

	// Build jobs for each process type
	for _, f := range formation {
		cmd := string(f.Command)
		env := environment(vars)

		// Build a Job for each instance of the process.
		for i := 1; i <= f.Count; i++ {
			j := &Job{
				Name:        NewName(name, version, f.ProcessType, i),
				Environment: env,
				Execute: Execute{
					Command: cmd,
					Image:   image,
				},
			}

			jobs = append(jobs, j)
		}
	}

	return jobs
}

// environment coerces a configs.Vars into a map[string]string.
func environment(vars configs.Vars) map[string]string {
	env := make(map[string]string)

	for k, v := range vars {
		env[string(k)] = string(v)
	}

	return env
}
