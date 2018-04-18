package empire

import (
	"fmt"
	"time"

	"golang.org/x/net/context"
)

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
	return nil, fmt.Errorf("`emp ps` is currently unsupported")
}
