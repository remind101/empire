package empire

import (
	"fmt"

	"github.com/remind101/empire/apps"
	"github.com/remind101/empire/configs"
	"github.com/remind101/empire/formations"
	"github.com/remind101/empire/images"
	"github.com/remind101/empire/processes"
	"github.com/remind101/empire/releases"
	"github.com/remind101/empire/scheduler"
)

// Manager is responsible for talking to the schedule to schedule jobs onto the
// cluster.
type Manager interface {
	// ScheduleRelease schedules a release onto the cluster.
	ScheduleRelease(*releases.Release) error
}

// manager provides a layer of convenience over a Scheduler.
type manager struct {
	scheduler.Scheduler
}

// NewService returns a new Service instance.
func NewManager(s scheduler.Scheduler) Manager {
	return &manager{
		Scheduler: s,
	}
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (s *manager) ScheduleRelease(release *releases.Release) error {
	jobs := buildJobs(
		release.App.Name,
		release.Version,
		*release.Slug.Image,
		release.Config.Vars,
		release.Formation,
	)

	return s.Scheduler.ScheduleMulti(jobs)
}

// newJobName returns a new Name with the proper format.
func newJobName(id apps.Name, v releases.Version, pt processes.Type, i int) scheduler.JobName {
	return scheduler.JobName(fmt.Sprintf("%s.%s.%s.%d", id, v, pt, i))
}

func buildJobs(name apps.Name, version releases.Version, image images.Image, vars configs.Vars, formation []*formations.CommandFormation) []*scheduler.Job {
	var jobs []*scheduler.Job

	// Build jobs for each process type
	for _, f := range formation {
		cmd := string(f.Command)
		env := environment(vars)

		// Build a Job for each instance of the process.
		for i := 1; i <= f.Count; i++ {
			j := &scheduler.Job{
				Name:        newJobName(name, version, f.ProcessType, i),
				Environment: env,
				Execute: scheduler.Execute{
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
