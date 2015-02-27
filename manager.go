package empire

import (
	"database/sql/driver"
	"fmt"

	"github.com/remind101/empire/scheduler"
)

type JobID string

// Scan implements the sql.Scanner interface.
func (id *JobID) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*id = JobID(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (id JobID) Value() (driver.Value, error) {
	return driver.Value(string(id)), nil
}

// Job represents a Job that was submitted to the scheduler.
type Job struct {
	ID JobID `db:"id"`

	AppName        `db:"app_id"`
	ReleaseVersion `db:"release_version"`
	ProcessType    `db:"process_type"`
	Instance       int `db:"instance"`

	Environment Vars    `db:"environment"`
	Image       Image   `db:"image"`
	Command     Command `db:"command"`
}

type JobState struct {
	Job       *Job
	MachineID string
	Name      scheduler.JobName
	State     string
}

func (j *Job) JobName() scheduler.JobName {
	return newJobName(
		j.AppName,
		j.ReleaseVersion,
		j.ProcessType,
		j.Instance,
	)
}

// JobQuery is a query object to filter results from JobsRepository List.
type JobQuery struct {
	App     AppName
	Release ReleaseVersion
}

// JobsRepository keeps track of all the Jobs that have been submitted to the
// scheduler.
type JobsRepository interface {
	Add(*Job) error
	Remove(*Job) error
	List(JobQuery) ([]*Job, error)
}

type jobsRepository struct {
	DB
}

func (r *jobsRepository) Add(job *Job) error {
	return r.DB.Insert(job)
}

func (r *jobsRepository) Remove(job *Job) error {
	_, err := r.DB.Exec(`delete from jobs where id = $1`, string(job.ID))
	return err
}

func (r *jobsRepository) List(q JobQuery) ([]*Job, error) {
	var jobs []*Job
	query := `select * from jobs where (app_id = $1 OR $1 = '') and (release_version = $2 OR $2 = 0)`
	return jobs, r.DB.Select(&jobs, query, string(q.App), int(q.Release))
}

// Manager is responsible for talking to the scheduler to schedule jobs onto the
// cluster.
type Manager interface {
	// ScheduleRelease schedules a release onto the cluster.
	ScheduleRelease(*Release, *Config, *Slug, Formation) error

	// ScaleRelease scales a release based on a process quantity map.
	ScaleRelease(*Release, *Config, *Slug, Formation, ProcessQuantityMap) error

	// FindJobsByApp returns JobStates for an app.
	JobStatesByApp(*App) ([]*JobState, error)
}

// manager is a base implementation of the Manager interface.
type manager struct {
	scheduler.Scheduler
	JobsRepository
}

// ScheduleRelease creates jobs for every process and instance count and
// schedules them onto the cluster.
func (m *manager) ScheduleRelease(release *Release, config *Config, slug *Slug, formation Formation) error {
	// Find any existing jobs that have been scheduled for this release.
	existing, err := m.existingJobs(release)
	if err != nil {
		return err
	}

	jobs := buildJobs(
		release.AppName,
		release.Ver,
		slug.Image,
		config.Vars,
		formation,
	)

	if len(existing) > len(jobs) {
		remove := existing[len(jobs):]

		if err := m.unscheduleMulti(remove); err != nil {
			return err
		}
	}

	return m.scheduleMulti(jobs)
}

func (m *manager) existingJobs(release *Release) ([]*Job, error) {
	return m.JobsRepository.List(JobQuery{
		App:     release.AppName,
		Release: release.Ver,
	})
}

func (m *manager) scheduleMulti(jobs []*Job) error {
	for _, j := range jobs {
		if err := m.schedule(j); err != nil {
			return err
		}
	}

	return nil
}

// schedule schedules a Job and adds it to the list of scheduled jobs.
func (m *manager) schedule(j *Job) error {
	name := j.JobName()
	env := environment(j.Environment)
	exec := scheduler.Execute{
		Command: string(j.Command),
		Image: scheduler.Image{
			Repo: string(j.Image.Repo),
			ID:   j.Image.ID,
		},
	}

	// Schedule the job onto the cluster.
	if err := m.Scheduler.Schedule(&scheduler.Job{
		Name:        name,
		Environment: env,
		Execute:     exec,
	}); err != nil {
		return err
	}

	// Add it to the list of scheduled jobs.
	if err := m.JobsRepository.Add(j); err != nil {
		return err
	}

	return nil
}

func (m *manager) unscheduleMulti(jobs []*Job) error {
	for _, j := range jobs {
		if err := m.unschedule(j); err != nil {
			return err
		}
	}

	return nil
}

func (m *manager) unschedule(j *Job) error {
	return m.Scheduler.Unschedule(j.JobName())
}

// ScaleRelease takes a release and process quantity map, and
// schedules/unschedules jobs to make the formation match the quantity map
func (m *manager) ScaleRelease(release *Release, config *Config, slug *Slug, formation Formation, qm ProcessQuantityMap) error {
	for t, q := range qm {
		if p, ok := formation[t]; ok {
			if err := m.scaleProcess(release, config, slug, t, p, q); err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *manager) scaleProcess(release *Release, config *Config, slug *Slug, t ProcessType, p *Process, q int) error {
	// Scale up
	if p.Quantity < q {
		for i := p.Quantity + 1; i <= q; i++ {
			err := m.schedule(
				&Job{
					AppName:        release.AppName,
					ReleaseVersion: release.Ver,
					ProcessType:    t,
					Instance:       i,
					Environment:    config.Vars,
					Image:          slug.Image,
					Command:        p.Command,
				},
			)
			if err != nil {
				return err
			}
		}
	}

	// Scale down
	if p.Quantity > q {
		for i := p.Quantity; i >= q; i-- {
			err := m.Scheduler.Unschedule(newJobName(release.AppName, release.Ver, t, i))
			if err != nil {
				return err
			}
		}
	}

	return nil
}

func (m *manager) JobStatesByApp(app *App) ([]*JobState, error) {
	// Jobs expected to be running
	jobs, err := m.JobsRepository.List(JobQuery{App: app.Name})
	if err != nil {
		return nil, err
	}

	// Job states for all existing jobs
	sjs, err := m.Scheduler.JobStates()
	if err != nil {
		return nil, err
	}

	// Create a map for easy lookups
	jsm := make(map[scheduler.JobName]*scheduler.JobState, len(sjs))
	for _, js := range sjs {
		jsm[js.Name] = js
	}

	// Create JobState based on Jobs and scheduler.JobStates
	js := make([]*JobState, len(jobs))
	for i, j := range jobs {
		s, ok := jsm[j.JobName()]

		machineID := "unknown"
		state := "unknown"
		if ok {
			machineID = s.MachineID
			state = s.State
		}

		js[i] = &JobState{
			Job:       j,
			Name:      j.JobName(),
			MachineID: machineID,
			State:     state,
		}
	}

	return js, nil
}

// newJobName returns a new Name with the proper format.
func newJobName(name AppName, v ReleaseVersion, t ProcessType, i int) scheduler.JobName {
	return scheduler.JobName(fmt.Sprintf("%s.%d.%s.%d", name, v, t, i))
}

func buildJobs(name AppName, version ReleaseVersion, image Image, vars Vars, f Formation) []*Job {
	var jobs []*Job

	// Build jobs for each process type
	for t, p := range f {
		// Build a Job for each instance of the process.
		for i := 1; i <= p.Quantity; i++ {
			j := &Job{
				AppName:        name,
				ReleaseVersion: version,
				ProcessType:    t,
				Instance:       i,
				Environment:    vars,
				Image:          image,
				Command:        p.Command,
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
