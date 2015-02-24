package empire

import (
	"fmt"

	"github.com/remind101/empire/scheduler"
)

// Manager is responsible for talking to the scheduler to schedule jobs onto the
// cluster.
type Manager interface {
	// ScheduleRelease schedules a release onto the cluster.
	ScheduleRelease(*Release) error
}

// manager is a base implementation of the Manager interface.
type manager struct {
	scheduler.Scheduler
}

// NewManager returns a new Service instance.
func NewManager(options Options) (Manager, error) {
	s, err := scheduler.NewScheduler(options.Fleet.API)
	if err != nil {
		return nil, err
	}

	return &manager{
		Scheduler: s,
	}, nil
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (s *manager) ScheduleRelease(release *Release) error {
	jobs := buildJobs(
		release.App.Name,
		release.Version,
		*release.Slug.Image,
		release.Config.Vars,
		release.Formation.Processes,
	)

	return s.Scheduler.ScheduleMulti(jobs)
}

// newJobName returns a new Name with the proper format.
func newJobName(name AppName, v ReleaseVersion, t ProcessType, i int) scheduler.JobName {
	return scheduler.JobName(fmt.Sprintf("%s.%s.%s.%d", name, v, t, i))
}

func buildJobs(name AppName, version ReleaseVersion, image Image, vars Vars, pm ProcessMap) []*scheduler.Job {
	var jobs []*scheduler.Job

	// Build jobs for each process type
	for t, p := range pm {
		cmd := string(p.Command)
		env := environment(vars)

		// Build a Job for each instance of the process.
		for i := 1; i <= p.Quantity; i++ {
			j := &scheduler.Job{
				Name:        newJobName(name, version, t, i),
				Environment: env,
				Execute: scheduler.Execute{
					Command: cmd,
					Image: scheduler.Image{
						Repo: string(image.Repo),
						ID:   image.ID,
					},
				},
			}

			jobs = append(jobs, j)
		}
	}

	return jobs
}

// environment coerces a Vars into a map[string]string.
func environment(vars Vars) map[string]string {
	env := make(map[string]string)

	for k, v := range vars {
		env[string(k)] = string(v)
	}

	return env
}
