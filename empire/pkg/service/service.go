// A Go package that provides a layer of abstraction above ECS to introduce the
// concept of Apps, which have are a collection of services.
package service

import (
	"time"

	"golang.org/x/net/context"
)

type Exposure int

const (
	ExposeNone Exposure = iota
	ExposePrivate
	ExposePublic
)

type App struct {
	// The name of the app.
	Name string

	// Process that belong to this app.
	Processes []*Process
}

type PortMap struct {
	// The Host port.
	Host *int64

	// The container port.
	Container *int64
}

type Process struct {
	// The type of process.
	Type string

	// The Image to run.
	Image string

	// The Command to run.
	Command string

	// Environment variables to set.
	Env map[string]string

	// Mapping of host -> container port mappings.
	Ports []PortMap

	// Exposure is the level of exposure for this process.
	Exposure Exposure

	// Instances is the desired instances of this service to run.
	Instances uint

	// The amount of RAM to allocate to this process in bytes.
	MemoryLimit uint

	// The amount of CPU to allocate to this process, out of 1024. Maps to
	// the --cpu-shares flag for docker.
	CPUShares uint

	// Attributes is an arbitrary map of attributes that can be assigned.
	Attributes map[string]interface{}
}

// Instance represents an Instance of a Process.
type Instance struct {
	Process *Process

	// The instance ID.
	ID string

	// The State that this Instance is in.
	State string

	// The time that this instance was last updated.
	UpdatedAt time.Time
}

// Manager is an interface for interfacing with Services.
type Manager interface {
	// Submit submits an app, creating it or updating it as necessary.
	Submit(context.Context, *App) error

	// Scale scales an app process.
	Scale(ctx context.Context, app string, process string, instances uint) error

	// Remove removes the App.
	Remove(ctx context.Context, app string) error

	// Instance lists the instances of a Process for an app.
	Instances(ctx context.Context, app string) ([]*Instance, error)

	// Stop stops an instance. The scheduler will automatically start a new
	// instance.
	Stop(ctx context.Context, instanceID string) error
}
