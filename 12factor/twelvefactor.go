// Package twelvefactor provides types to represents 12factor applications,
// which are defined in http://12factor.net/
package twelvefactor

import (
	"io"
	"time"
)

// Manifest describes a 12factor application and it's processes.
type Manifest struct {
	App
	Processes []Process
}

// App represents a 12factor application. We define an application as a
// collection of processes that share a common environment and container image.
type App struct {
	// Unique identifier of the application.
	ID string

	// Name of the application.
	Name string

	// A string representing the version of this App.
	Version string

	// The container image for this app.
	Image string

	// The shared environment variables for the individual processes.
	Env map[string]string
}

// Process represents an individual Process of an App, which defines the command
// to run within the container image.
type Process struct {
	// A unique identifier for this process, within the scope of the app.
	// Generally this would be something like "web" or "worker.
	Name string

	// Exposure is used by schedulers to determine if the process exposes any
	// TCP/HTTP/HTTPS services. Schedulers can use the Protocol method or
	// perform a type assertion to determine the exposure and settings for
	// the exposure.
	Exposure Exposure

	// The command to run when running this process.
	Command []string

	// Additional environment variables to merge with the App's environment
	// when running this process.
	Env map[string]string

	// Free form labels to attach to this process.
	Labels map[string]string

	// Where Stdout for this process should go to.
	Stdout io.Writer

	// Where Stdin for this process should come from. The zero value is to
	// not attach Stdin.
	Stdin io.Reader

	// The desired number of instances to run.
	DesiredCount int

	// The amount of memory to allocate to this process, in bytes.
	Memory int

	// The number of CPU Shares to allocate to this process.
	CPUShares int
}

// Task represents the state of an individual instance of a Process.
type Task struct {
	// A globally unique identifier for this task.
	ID string

	// The app version that this task relates to.
	Version string

	// The process that this task relates to.
	Process string

	// The state that this task is in.
	State string

	// The time that this state was recorded at.
	Time time.Time
}

// Exposure represents a service that a process exposes, like HTTP/HTTPS/TCP or
// SSL.
type Exposure interface {
	Protocol() string
}

var (
	// The zero value for Exposure. No services.
	ExposeNone Exposure = nil
)

// HTTPExposure represents an HTTP exposure.
type HTTPExposure struct {
	// External means that this http service should be exposed to internal
	// traffic if possible.
	External bool
}

func (e *HTTPExposure) Protocol() string { return "http" }

// HTTPSExposure represents an HTTPS exposure
type HTTPSExposure struct {
	HTTPExposure

	// The certificate to attach to the process.
	Cert string
}

func (e *HTTPSExposure) Protocol() string { return "https" }

// ProcessEnv merges the App environment with any environment variables provided
// in the process.
func ProcessEnv(app App, process Process) map[string]string {
	return MergeEnv(app.Env, process.Env)
}

// Merges the maps together, favoring keys from the right to the left.
func MergeEnv(envs ...map[string]string) map[string]string {
	merged := make(map[string]string)
	for _, env := range envs {
		for k, v := range env {
			merged[k] = v
		}
	}
	return merged
}
