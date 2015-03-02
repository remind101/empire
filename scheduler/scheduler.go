package scheduler

// ContainerName represents the (unique) name of a container.
type ContainerName string

// Image represents a container image, which is tied to a repository.
type Image struct {
	Repo string
	ID   string
}

// Container represents a container to schedule onto the cluster.
type Container struct {
	// The unique name of the container.
	Name ContainerName

	// A map of environment variables to set.
	Environment map[string]string

	// The command to run.
	Command string

	// The image that this container will be built from.
	Image Image
}

// State represents the state of a container.
type State int

// Various states that a container can be in.
const (
	StatePending State = iota
	StateRunning
	StateFailed
)

// ContainerState represents the status of a container.
type ContainerState struct {
	MachineID string
	Name      ContainerName
	State     string // TODO use State type
}

// Scheduler is an interface that represents something that can schedule Jobs
// onto the cluster.
type Scheduler interface {
	// Schedule schedules a container to run on the cluster.
	Schedule(*Container) error

	// Unschedule unschedules a container from the cluster by its name.
	Unschedule(ContainerName) error

	// List ContainerStates
	ContainerStates() ([]*ContainerState, error)
}

// NewScheduler is a factory method for generating a new Scheduler instance.
func NewScheduler(fleet string) (Scheduler, error) {
	if fleet == "" {
		return newScheduler(), nil
	}

	return newFleetScheduler(fleet)
}

// scheduler is a fake implementation of the Scheduler interface.
type scheduler struct{}

func newScheduler() *scheduler {
	return &scheduler{}
}

// Schedule implements Scheduler Schedule.
func (s *scheduler) Schedule(c *Container) error {
	return nil
}

// Unschedule implements Scheduler Unschedule.
func (s *scheduler) Unschedule(n ContainerName) error {
	return nil
}

func (s *scheduler) ContainerStates() ([]*ContainerState, error) {
	return nil, nil
}
