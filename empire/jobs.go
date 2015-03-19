package empire

import (
	"fmt"
	"time"

	"github.com/remind101/empire/empire/pkg/container"
	"github.com/remind101/empire/empire/pkg/logger"
	"golang.org/x/net/context"
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

func (s *store) JobsList(q JobsListQuery) ([]*Job, error) {
	return jobsList(s.db, q)
}

func (s *store) JobsCreate(job *Job) (*Job, error) {
	return jobsCreate(s.db, job)
}

func (s *store) JobsDestroy(job *Job) error {
	return jobsDestroy(s.db, job)
}

type jobsService struct {
	store     *store
	scheduler container.Scheduler
}

func (s *jobsService) Schedule(ctx context.Context, jobs ...*Job) error {
	for _, j := range jobs {
		logger.Log(ctx,
			"at", "job.schedule",
			"app", j.AppName,
			"release", j.ReleaseVersion,
			"process", j.ProcessType,
			"instance", j.Instance,
		)
		if _, err := schedule(s.store, s.scheduler, j); err != nil {
			return err
		}
	}

	return nil
}

func (s *jobsService) Unschedule(ctx context.Context, jobs ...*Job) error {
	for _, j := range jobs {
		logger.Log(ctx,
			"at", "job.unschedule",
			"app", j.AppName,
			"release", j.ReleaseVersion,
			"process", j.ProcessType,
			"instance", j.Instance,
		)
		if err := unschedule(s.store, s.scheduler, j); err != nil {
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

// schedule schedules to job onto the cluster, then persists it to the database.
func schedule(store *store, s container.Scheduler, j *Job) (*Job, error) {
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

	return store.JobsCreate(j)
}

func unschedule(store *store, s container.Scheduler, j *Job) error {
	if err := s.Unschedule(j.ContainerName()); err != nil {
		return err
	}

	return store.JobsDestroy(j)
}

type jobStatesService struct {
	store     *store
	scheduler container.Scheduler
}

func (s *jobStatesService) JobStatesByApp(app *App) ([]*JobState, error) {
	// Jobs expected to be running
	jobs, err := s.store.JobsList(JobsListQuery{
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
