package empire

import (
	"io"
	"time"

	"golang.org/x/net/context"
)

type IO struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
}

type TaskEngine interface {
	Run(context.Context, *App, *IO) error
	Tasks(context.Context, *App) ([]*Task, error)
	Stop(context.Context, string) error
}

// Host represents the host of the task
type Host struct {
	// the host id
	ID string
}

// Task represents a running process.
type Task struct {
	// The name of the task.
	Name string

	// The name of the process that this task is for.
	Type string

	// The task id
	ID string

	// The host of the task
	Host Host

	// The command that this task is running.
	Command Command

	// The state of the task.
	State string

	// The time that the state was recorded.
	UpdatedAt time.Time

	// The constraints of the Process.
	Constraints Constraints
}

type tasksService struct {
	*Empire
}

func (s *tasksService) Tasks(ctx context.Context, app *App) ([]*Task, error) {
	return s.engine.Tasks(ctx, app)
}
