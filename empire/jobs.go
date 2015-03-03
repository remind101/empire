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

func NewJobsRepository(db DB) (JobsRepository, error) {
	return &jobsRepository{db}, nil
}

type jobsRepository struct {
	DB
}

func (r *jobsRepository) Add(job *Job) error {
	_, err := CreateJob(r.DB, job)
	return err
}

func (r *jobsRepository) Remove(job *Job) error {
	return DestroyJob(r.DB, job)
}

func (r *jobsRepository) List(q JobQuery) ([]*Job, error) {
	return ListJobs(r.DB, q)
}

// CreateJob inserts the Job into the database.
func CreateJob(db Inserter, job *Job) (*Job, error) {
	return job, db.Insert(job)
}

// DestroyJob removes a Job from the database.
func DestroyJob(db Deleter, job *Job) error {
	_, err := db.Delete(job)
	return err
}

// ListJobs returns a filtered list of Jobs.
func ListJobs(db Queryier, q JobQuery) ([]*Job, error) {
	var jobs []*Job
	query := `select * from jobs where (app_id = $1 OR $1 = '') and (release_version = $2 OR $2 = 0)`
	return jobs, db.Select(&jobs, query, string(q.App), int(q.Release))
}

// Schedule is an interface that represents something that can schedule jobs
// onto the cluster.
type Scheduler interface {
	Schedule(...*Job) error
	Unschedule(...*Job) error
}

type JobsService interface {
	JobsRepository
	Scheduler

	// FindJobsByApp returns JobStates for an app.
	JobStatesByApp(*App) ([]*JobState, error)

	// Find existing jobs by app name.
	JobsByApp(AppName) ([]*Job, error)
}

type jobsService struct {
	JobsRepository
	scheduler.Scheduler
}

func (s *jobsService) JobStatesByApp(app *App) ([]*JobState, error) {
	// Jobs expected to be running
	jobs, err := s.JobsByApp(app.Name)
	if err != nil {
		return nil, err
	}

	// Job states for all existing jobs
	sjs, err := s.Scheduler.JobStates()
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

func (s *jobsService) JobsByApp(appName AppName) ([]*Job, error) {
	return s.JobsRepository.List(JobQuery{
		App: appName,
	})
}

func (s *jobsService) Schedule(jobs ...*Job) error {
	for _, j := range jobs {
		if err := s.schedule(j); err != nil {
			return err
		}
	}

	return nil
}

// schedule schedules a Job and adds it to the list of scheduled jobs.
func (s *jobsService) schedule(j *Job) error {
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
	if err := s.Scheduler.Schedule(&scheduler.Job{
		Name:        name,
		Service:     fmt.Sprintf("%s/%s", j.ProcessType, j.AppName), // Used for registrator integration
		Environment: env,
		Execute:     exec,
	}); err != nil {
		return err
	}

	// Add it to the list of scheduled jobs.
	if err := s.JobsRepository.Add(j); err != nil {
		return err
	}

	return nil
}

func (s *jobsService) Unschedule(jobs ...*Job) error {
	for _, j := range jobs {
		if err := s.unschedule(j); err != nil {
			return err
		}
	}

	return nil
}

func (s *jobsService) unschedule(j *Job) error {
	err := s.Scheduler.Unschedule(j.JobName())
	if err != nil {
		return err
	}

	return s.JobsRepository.Remove(j)
}
