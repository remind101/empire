// Package scheduler provides the core interface that Empire uses when
// interacting with a cluster of machines to run tasks.
package twelvefactor

import (
	"io"
	"time"

	"golang.org/x/net/context"

	"github.com/remind101/empire/pkg/image"
)

type Manifest struct {
	// The id of the app.
	AppID string

	// An identifier that represents the version of this release.
	Release string

	// The name of the app.
	Name string

	// The application environment.
	Env map[string]string

	// The application labels.
	Labels map[string]string

	// Process that belong to this app.
	Processes []*Process
}

type Process struct {
	// The type of process.
	Type string

	// The Image to run.
	Image image.Image

	// The Command to run.
	Command []string

	// Environment variables to set.
	Env map[string]string

	// Labels to set on the container.
	Labels map[string]string

	// The amount of RAM to allocate to this process in bytes.
	Memory uint

	// The amount of CPU to allocate to this process, out of 1024. Maps to
	// the --cpu-shares flag for docker.
	CPUShares uint

	// ulimit -u
	Nproc uint

	// Quantity is the desired instances of this service to run.
	Quantity int

	// Exposure is the level of exposure for this process.
	Exposure *Exposure

	// Can be used to setup a CRON schedule to run this task periodically.
	Schedule Schedule
}

// Schedule represents a Schedule for scheduled tasks that run periodically.
type Schedule interface{}

// CRONSchedule is a Schedule implementation that represents a CRON expression.
type CRONSchedule string

// Exposure controls the exposure settings for a process.
type Exposure struct {
	// External means that this process will be exposed to internet facing
	// traffic, as opposed to being internal. How this is used is
	// implementation specific. For ECS, this means that the attached ELB
	// will be "internet-facing".
	External bool

	// The ports to expose and map to the container.
	Ports []Port
}

// Port maps a host port to a container port.
type Port struct {
	// The port that external applications will connect to. It's
	// implementation specific as to what this is used for. For example,
	// with ECS, this is used as the LoadBalancerPort.
	Host int

	// The port within the container that the process should bind to.
	Container int

	// The exposure type (e.g. HTTPExposure, HTTPSExposure, TCPExposure).
	Protocol Protocol
}

// Protocol represents a service that a process exposes, like HTTP/HTTPS/TCP or
// SSL.
type Protocol interface {
	Protocol() string
}

// HTTP represents an HTTP exposure.
type HTTP struct{}

func (e *HTTP) Protocol() string { return "http" }

// TCP represents a tcp exposure.
type TCP struct{}

func (e *TCP) Protocol() string { return "tcp" }

// HTTPS represents an HTTPS exposure
type HTTPS struct {
	// The certificate to attach to the process.
	Cert string
}

func (e *HTTPS) Protocol() string { return "https" }

// SSL represents a secure TCP exposure
type SSL struct {
	// The certificate to attach to the process.
	Cert string
}

func (e *SSL) Protocol() string { return "ssl" }

// Host represents the host of an instance
type Host struct {
	// The host ID.
	ID string
}

// Task represents an Task of a Process.
type Task struct {
	Process *Process

	// The instance ID.
	ID string

	// The instance host
	Host Host

	// The State that this Instance is in.
	State string

	// The time that this instance was last updated.
	UpdatedAt time.Time
}

type Runner interface {
	// Run runs a process.
	Run(ctx context.Context, app *Manifest, process *Process, in io.Reader, out io.Writer) error
}

// Scheduler is an interface for interfacing with Services.
type Scheduler interface {
	Runner

	// Submit submits an app, creating it or updating it as necessary.
	// When StatusStream is nil, Submit should return as quickly as possible,
	// usually when the new version has been received, and validated. If
	// StatusStream is not nil, it's recommended that the method not return until
	// the deployment has fully completed.
	Submit(context.Context, *Manifest, StatusStream) error

	// Remove removes the App.
	Remove(ctx context.Context, app string) error

	// Instance lists the instances of a Process for an app.
	Tasks(ctx context.Context, app string) ([]*Task, error)

	// Stop stops an instance. The scheduler will automatically start a new
	// instance.
	Stop(ctx context.Context, instanceID string) error

	// Restart restarts the processes within the App.
	Restart(context.Context, *Manifest, StatusStream) error
}

// Env merges the App environment with any environment variables provided
// in the process.
func Env(app *Manifest, process *Process) map[string]string {
	return merge(app.Env, process.Env)
}

// Labels merges the App labels with any labels provided in the process.
func Labels(app *Manifest, process *Process) map[string]string {
	return merge(app.Labels, process.Labels)
}

// merges the maps together, favoring keys from the right to the left.
func merge(envs ...map[string]string) map[string]string {
	merged := make(map[string]string)
	for _, env := range envs {
		for k, v := range env {
			merged[k] = v
		}
	}
	return merged
}

type Status struct {
	// A friendly human readable message about the status change.
	Message string
}

// String implements the fmt.Stringer interface.
func (s *Status) String() string {
	return s.Message
}

// StatusStream is an interface for publishing status updates while a scheduler
// is executing.
type StatusStream interface {
	// Publish publishes an update to the status stream
	Publish(Status) error
}

// StatusStreamFunc is a function that implements the Statusstream interface
type StatusStreamFunc func(Status) error

func (fn StatusStreamFunc) Publish(status Status) error {
	return fn(status)
}

// NullStatusStream a status stream that does nothing.
var NullStatusStream = StatusStreamFunc(func(status Status) error {
	return nil
})
