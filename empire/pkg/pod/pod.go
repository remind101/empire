// package pod provides a level of abstraction above pacakge container, allowing
// consumers to schedule collections of containers.
package pod

import (
	"time"

	"github.com/remind101/empire/empire/pkg/container"
)

type Image struct {
	Repo string
	ID   string
}

// Templates represents a template for running a container.
type Template struct {
	// ID is a unique identifier for this Template.
	ID string

	// Environment variables to set in the container
	Env map[string]string

	// The command to run.
	Command string

	// The image to create the container from.
	Image Image

	// Tags are arbitrary metadata to tag onto the pod.
	Tags map[string]string

	// Instances controls how many instances of the Template to maintain.
	Instances uint
}

// Instance represents a running instance of a Template.
type Instance struct {
	Template *Template `json:"-"`

	// The instance number for this Instance.
	Instance uint

	// When this instance was created.
	CreatedAt time.Time
}

// newInstance returns a new Instance for the given Template.
func newInstance(template *Template, instance uint) *Instance {
	return &Instance{
		Template: template,
		Instance: instance,
	}
}

// InstanceState represents the state of a given Template Instance.
type InstanceState struct {
	// Associated Instance.
	*Instance

	// The state of the instance, as provided from the scheduler.
	State string

	// When this state was last updated.
	UpdatedAt time.Time
}

// newInstanceState returns a new InstanceState instance for an instance, based
// on a ContainerState.
func newInstanceState(instance *Instance, containerState *container.ContainerState) *InstanceState {
	state := "unknown"

	if containerState != nil {
		state = containerState.State
	}

	return &InstanceState{
		Instance:  instance,
		State:     state,
		UpdatedAt: instance.CreatedAt,
	}
}
