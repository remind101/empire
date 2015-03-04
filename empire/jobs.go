package empire

import (
	"database/sql/driver"
	"fmt"
	"time"

	"github.com/remind101/empire/empire/scheduler"
	"gopkg.in/gorp.v1"
)

// JobID represents a unique identifier for a Job.
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

	Environment Vars  `db:"environment"`
	Image       Image `db:"image"`
	Command     `db:"command"`

	// UpdatedAt indicates when this job last changed state.
	UpdatedAt time.Time `db:"updated_at"`
}

// PreInsert implements a pre insert hook for the db interface
func (j *Job) PreInsert(s gorp.SqlExecutor) error {
	j.UpdatedAt = Now()
	return nil
}

func (j *Job) JobName() scheduler.JobName {
	return newJobName(
		j.AppName,
		j.ReleaseVersion,
		j.ProcessType,
		j.Instance,
	)
}

// JobState represents the state of a submitted job.
type JobState struct {
	Job       *Job
	MachineID string
	Name      scheduler.JobName
	State     string
}

// Schedule is an interface that represents something that can schedule jobs
// onto the cluster.
type Scheduler interface {
	Schedule(...*Job) error
	Unschedule(...*Job) error
}

type JobsFinder interface {
	JobsList(JobsListQuery) ([]*Job, error)
}

type JobsService interface {
	Scheduler
	JobsFinder
}

type jobsService struct {
	DB
	scheduler scheduler.Scheduler
}

func (s *jobsService) JobsList(q JobsListQuery) ([]*Job, error) {
	return JobsList(s.DB, q)
}

func (s *jobsService) Schedule(jobs ...*Job) error {
	for _, j := range jobs {
		if _, err := Schedule(s.DB, s.scheduler, j); err != nil {
			return err
		}
	}

	return nil
}

func (s *jobsService) Unschedule(jobs ...*Job) error {
	for _, j := range jobs {
		if err := Unschedule(s.DB, s.scheduler, j); err != nil {
			return err
		}
	}

	return nil
}

// JobsCreate inserts the Job into the database.
func JobsCreate(db Inserter, job *Job) (*Job, error) {
	return job, db.Insert(job)
}

// JobsDestroy removes a Job from the database.
func JobsDestroy(db Deleter, job *Job) error {
	_, err := db.Delete(job)
	return err
}

// JobsListQuery is a query object to filter results from JobsRepository List.
type JobsListQuery struct {
	App     AppName
	Release ReleaseVersion
}

// JobsList returns a filtered list of Jobs.
func JobsList(db Queryier, q JobsListQuery) ([]*Job, error) {
	var jobs []*Job
	query := `select * from jobs where (app_id = $1 OR $1 = '') and (release_version = $2 OR $2 = 0)`
	return jobs, db.Select(&jobs, query, string(q.App), int(q.Release))
}

// Schedule schedules to job onto the cluster, then persists it to the database.
func Schedule(db Inserter, s scheduler.Scheduler, j *Job) (*Job, error) {
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
	if err := s.Schedule(&scheduler.Job{
		Name:        name,
		Service:     fmt.Sprintf("%s/%s", j.ProcessType, j.AppName), // Used for registrator integration
		Environment: env,
		Execute:     exec,
	}); err != nil {
		return nil, err
	}

	return JobsCreate(db, j)
}

func Unschedule(db Deleter, s scheduler.Scheduler, j *Job) error {
	if err := s.Unschedule(j.JobName()); err != nil {
		return err
	}

	return JobsDestroy(db, j)
}

type JobStatesFinder interface {
	JobStatesByApp(*App) ([]*JobState, error)
}

type JobStatesService interface {
	JobStatesFinder
}

type jobStatesService struct {
	DB
	JobsService
	scheduler scheduler.Scheduler
}

func (s *jobStatesService) JobStatesByApp(app *App) ([]*JobState, error) {
	// Jobs expected to be running
	jobs, err := s.JobsService.JobsList(JobsListQuery{
		App: app.Name,
	})
	if err != nil {
		return nil, err
	}

	// Job states for all existing jobs
	sjs, err := s.scheduler.JobStates()
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
