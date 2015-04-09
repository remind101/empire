// package container is a Go package that provides a common abstraction for
// scheduling containers onto a compute cluster.
package container

import "fmt"

// Image represents a containerized image. This could be a Docker image, a
// rocket image, etc.
type Image struct {
	Repo string
	ID   string
}

// String implements the fmt.Stringer interface.
func (i Image) String() string {
	return fmt.Sprintf("%s:%s", i.Repo, i.ID)
}

// Container represents a container.
type Container struct {
	// The name of the container
	Name string

	// Environment variables to set in the container
	Env map[string]string

	// The command to run.
	Command string

	// The image to create the container from.
	Image Image

	// Memory limit in docker run fmt: <number><optional unit>, where unit = b, k, m or g
	MemoryLimit string
}

// ContainerState represents the state of a scheduled container.
type ContainerState struct {
	*Container

	// State represents the current state of this container.
	State string

	// MachineID represents the machine that the container is scheduled
	// onto.
	MachineID string
}
