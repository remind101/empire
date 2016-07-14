package empire

import (
	"time"

	"github.com/remind101/empire/pkg/constraints"
	"github.com/remind101/empire/scheduler"
	"golang.org/x/net/context"
)

// Task represents a running process.
type Task struct {
	// The id of the task.
	ID string

	// The release that this task relates to.
	Release string

	// The name of the process that this task is for.
	Type string

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
	var tasks []*Task

	instances, err := s.Scheduler.Instances(ctx, app.ID)
	if err != nil {
		return tasks, err
	}

	for _, i := range instances {
		tasks = append(tasks, taskFromInstance(i))
	}

	return tasks, nil
}

// taskFromInstance converts a scheduler.Instance into a Task.
// It pulls some of its data from empire specific environment variables if they have been set.
// Once ECS supports this data natively, we can stop doing this.
func taskFromInstance(i *scheduler.Instance) *Task {
	version := i.Process.Env["EMPIRE_RELEASE"]
	if version == "" {
		version = "v0"
	}

	return &Task{
		ID:      i.ID,
		Release: version,
		Type:    string(i.Process.Type),
		Command: Command(i.Process.Command),
		Constraints: Constraints{
			CPUShare: constraints.CPUShare(i.Process.CPUShares),
			Memory:   constraints.Memory(i.Process.MemoryLimit),
			Nproc:    constraints.Nproc(i.Process.Nproc),
		},
		State:     i.State,
		UpdatedAt: i.UpdatedAt,
	}
}
