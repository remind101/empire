package twelvefactor

// Runner is an interface that wraps the basic Run method, providing a way to
// run a 12factor application.
//
// Implementors should ensure that any existing processes that don't exist in
// the newly submitted process list are removed. For example, if a "web" process
// was previously defined, then only a "worker" process was submitted, the
// existing "web" process should be removed.
type Runner interface {
	Run(Manifest) error
}

// ProcessRunner is an optional interface for running attached and detached
// processes. This is usefuly for running attached processes like a rails
// console or detached processes like database migrations.
//
// Attached vs Detached is determined from the Stdout stream.
type ProcessRunner interface {
	RunProcess(app string, process Process) error
}

// ProcessScaler is an interface that wraps the basic Scale method for scaling a
// process by name for an application.
type ProcessScaler interface {
	ScaleProcess(app, process string, desired int) error
}

// Remover is an interface that wraps the basic Remove method for removing an
// app and all of it's processes.
type Remover interface {
	Remove(app string) error
}

// Restarter is an interface that wraps the Restart method, which provides a
// method for restarting an App.
type Restarter interface {
	Restart(app string) error
}

// ProcessRestarter is an interface that wraps the RestartProcess method, which
// provides a method for restarting a Process.
type ProcessRestarter interface {
	RestartProcess(app string, process string) error
}

// Scheduler provides an interface for running twelve factor applications.
type Scheduler interface {
	Runner
	Remover
	ProcessScaler
	Restarter
	ProcessRestarter

	// Returns the tasks for the given application.
	Tasks(app string) ([]Task, error)

	// Stops an individual task.
	StopTask(taskID string) error
}
