package container

// Scheduler defines our interface that implementations must conform to.
type Scheduler interface {
	// Schedule schedules one or more containers to run on the cluster.
	Schedule(...*Container) error

	// Unschedule unschedules one or more containers, by name.
	Unschedule(...string) error

	// ContainerStates returns a slice of ContainerState entities that
	// represent the state of all Containers scheduled onto the cluster.
	ContainerStates() ([]*ContainerState, error)
}
