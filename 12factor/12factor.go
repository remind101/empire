// Package twelvefactor provides types to represents 12factor applications,
// which are defined in http://12factor.net/
package twelvefactor

import "github.com/remind101/empire/pkg/image"

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

	// The container image for this app.
	Image image.Image

	// The shared environment variables for the individual processes.
	Env map[string]string

	// The shared labels for the individual processes.
	Labels map[string]string

	Processes []Process
}

// Process represents an individual Process of an App, which defines the command
// to run within the container image.
type Process struct {
	// A unique identifier for this process, within the scope of the app.
	// Generally this would be something like "web" or "worker.
	Type string

	// Exposure is used by schedulers to determine if the process exposes any
	// TCP/HTTP/HTTPS services. Schedulers can use the Protocol method or
	// perform a type assertion to determine the exposure and settings for
	// the exposure.
	Exposure *Exposure

	// The command to run when running this process.
	Command []string

	// Additional environment variables to merge with the App's environment
	// when running this process.
	Env map[string]string

	// Free form labels to attach to this process.
	Labels map[string]string

	// The desired number of instances to run.
	Instances uint

	// The amount of memory to allocate to this process, in bytes.
	MemoryLimit uint

	// The number of CPU Shares to allocate to this process.
	CPUShares uint

	// ulimit -u
	Nproc uint
}

// Exposure controls the exposure settings for a process.
type Exposure struct {
	// External means that this process will be exposed to internet facing
	// traffic, as opposed to being internal. How this is used is
	// implementation specific. For ECS, this means that the attached ELB
	// will be "internet-facing".
	External bool

	// The exposure type (e.g. HTTPExposure, HTTPSExposure, TCPExposure).
	Type ExposureType
}

// Exposure represents a service that a process exposes, like HTTP/HTTPS/TCP or
// SSL.
type ExposureType interface {
	Protocol() string
}

// HTTPExposure represents an HTTP exposure.
type HTTPExposure struct{}

func (e *HTTPExposure) Protocol() string { return "http" }

// HTTPSExposure represents an HTTPS exposure
type HTTPSExposure struct {
	// The certificate to attach to the process.
	Cert string
}

func (e *HTTPSExposure) Protocol() string { return "https" }

// ProcessEnv merges the App environment with any environment variables provided
// in the process.
func ProcessEnv(app App, process Process) map[string]string {
	return merge(app.Env, process.Env)
}

// ProcessLabels merges the App labels with any labels provided in the process.
func ProcessLabels(app App, process Process) map[string]string {
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
