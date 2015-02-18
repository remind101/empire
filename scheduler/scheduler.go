package scheduler

import "github.com/remind101/empire/manager"

// Job is a job that can be scheduled.
type Job struct {
	// The unique name of the job.
	Name string

	// A map of environment variables to set.
	Environment map[string]string

	// The command to run.
	Execute manager.Execute

	// Meta data useful for querying with
	Meta map[string]string
}

type JobMap map[string]Job

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

// Query is used to specify filtering conditions for Job queries
type Query struct {
	// Select Jobs that contain this map of keys and values
	// in their meta data.
	Meta map[string]string
}

// Scheduler is an interface for scheduling Jobs
type Scheduler interface {
	Schedule(*Job) error
	Unschedule(*Job) error
	Jobs(*Query) (JobMap, error)
}
