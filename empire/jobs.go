package empire

import (
	"fmt"
	"time"

	"github.com/remind101/empire/empire/pkg/container"
	"gopkg.in/gorp.v1"
)

// Job represents a Job that was submitted to the scheduler.
type Job struct {
	ID string `db:"id"`

	AppName        string `db:"app_id"`
	ReleaseVersion int    `db:"release_version"`
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

func (j *Job) ContainerName() string {
	return newContainerName(
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
	Name      string
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
	*db
	scheduler container.Scheduler
}

func (s *jobsService) JobsList(q JobsListQuery) ([]*Job, error) {
	return jobsList(s.db, q)
}

func (s *jobsService) Schedule(jobs ...*Job) error {
	for _, j := range jobs {
		if _, err := Schedule(s.db, s.scheduler, j); err != nil {
			return err
		}
	}

	return nil
}

func (s *jobsService) Unschedule(jobs ...*Job) error {
	for _, j := range jobs {
		if err := unschedule(s.db, s.scheduler, j); err != nil {
			return err
		}
	}

	return nil
}

// JobsCreate inserts the Job into the database.
func jobsCreate(db *db, job *Job) (*Job, error) {
	return job, db.Insert(job)
}

// JobsDestroy removes a Job from the database.
func jobsDestroy(db *db, job *Job) error {
	_, err := db.Delete(job)
	return err
}

// JobsListQuery is a query object to filter results from JobsRepository List.
type JobsListQuery struct {
	App     string
	Release int
}

// JobsList returns a filtered list of Jobs.
func jobsList(db *db, q JobsListQuery) ([]*Job, error) {
	var jobs []*Job
	query := `select * from jobs where (app_id = $1 OR $1 = '') and (release_version = $2 OR $2 = 0)`
	return jobs, db.Select(&jobs, query, string(q.App), int(q.Release))
}

// Schedule schedules to job onto the cluster, then persists it to the database.
func Schedule(db *db, s container.Scheduler, j *Job) (*Job, error) {
	env := environment(j.Environment)
	env["SERVICE_NAME"] = fmt.Sprintf("%s/%s", j.ProcessType, j.AppName)

	container := &container.Container{
		Name:    j.ContainerName(),
		Env:     env,
		Command: string(j.Command),
		Image: container.Image{
			Repo: string(j.Image.Repo),
			ID:   j.Image.ID,
		},
	}

	// Schedule the container onto the cluster.
	if err := s.Schedule(container); err != nil {
		return nil, err
	}

	return jobsCreate(db, j)
}

func unschedule(db *db, s container.Scheduler, j *Job) error {
	if err := s.Unschedule(j.ContainerName()); err != nil {
		return err
	}

	return jobsDestroy(db, j)
}

type JobStatesFinder interface {
	JobStatesByApp(*App) ([]*JobState, error)
}

type JobStatesService interface {
	JobStatesFinder
}

type jobStatesService struct {
	*db
	JobsService
	scheduler container.Scheduler
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
	sjs, err := s.scheduler.ContainerStates()
	if err != nil {
		return nil, err
	}

	// Create a map for easy lookups
	jsm := make(map[string]*container.ContainerState, len(sjs))
	for _, js := range sjs {
		jsm[js.Name] = js
	}

	// Create JobState based on Jobs and container.ContainerStates
	js := make([]*JobState, len(jobs))
	for i, j := range jobs {
		s, ok := jsm[j.ContainerName()]

		machineID := "unknown"
		state := "unknown"
		if ok {
			machineID = s.MachineID
			state = s.State
		}

		js[i] = &JobState{
			Job:       j,
			Name:      j.ContainerName(),
			MachineID: machineID,
			State:     state,
		}
	}

	return js, nil
}
