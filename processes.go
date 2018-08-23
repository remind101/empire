package empire

import (
	"errors"
	"fmt"
	"strings"

	"github.com/remind101/empire/internal/shellwords"
	"github.com/remind101/empire/pkg/constraints"
)

// DefaultQuantities maps a process type to the default number of instances to
// run.
var DefaultQuantities = map[string]int{
	"web": 1,
}

const (
	// webProcessType is the process type we assume are web server processes.
	webProcessType = "web"
)

// DefaultWebPort is added to "web" procs that don't have explicit ports.
var DefaultWebPort = Port{
	Host:      80,
	Container: 8080,
	Protocol:  "http",
}

// Command represents a command and it's arguments. For example:
type Command []string

// ParseCommand parses a string into a Command, taking quotes and other shell
// words into account. For example:
func ParseCommand(command string) (Command, error) {
	return shellwords.Parse(command)
}

// MustParseCommand parses the string into a Command, panicing if there's an
// error. This method should only be used in tests for convenience.
func MustParseCommand(command string) Command {
	c, err := ParseCommand(command)
	if err != nil {
		panic(err)
	}
	return c
}

// String returns the string reprsentation of the command.
func (c Command) String() string {
	return strings.Join([]string(c), " ")
}

// Process holds configuration information about a Process.
type Process struct {
	// Command is the command to run.
	Command Command `json:"command,omitempty"`

	// Port mappings from container to load balancer.
	Ports []Port `json:"ports,omitempty"`

	// Signifies that this is a named one off command and not a long lived
	// service.
	NoService bool `json:"no_service,omitempty"`

	// Quantity is the desired number of instances of this process.
	Quantity int `json:"quantity,omitempty"`

	// The memory constraints, in bytes.
	Memory constraints.Memory `json:"memory,omitempty"`

	// The amount of CPU share to give.
	CPUShare constraints.CPUShare `json:"cpu_share,omitempty"`

	// The allow number of unix processes within the container.
	Ulimits []Ulimit `json:"ulimits,omitempty"`

	// A cron expression. If provided, the process will be run as a
	// scheduled task.
	Cron *string `json:"cron,omitempty"`

	// Any process specific environment variables.
	Environment map[string]string `json:"environment,omitempty"`
}

type Ulimit struct {
	Name      string `json:"name"`
	SoftLimit int    `json:"soft_limit"`
	HardLimit int    `json:"hard_limit"`
}

type Port struct {
	Host      int    `json:"host"`
	Container int    `json:"container"`
	Protocol  string `json:"protocol"`
}

// IsValid returns nil if the Process is valid.
func (p *Process) IsValid() error {
	// Ensure that processes marked as NoService can't be scaled up.
	if p.NoService {
		if p.Quantity != 0 {
			return errors.New("non-service processes cannot be scaled up")
		}
	}

	return nil
}

// Constraints returns a constraints.Constraints from this Process definition.
func (p *Process) Constraints() Constraints {
	return Constraints{
		Memory:   p.Memory,
		CPUShare: p.CPUShare,
	}
}

// SetConstraints sets the memory/cpu/nproc for this Process to the given
// constraints.
func (p *Process) SetConstraints(c Constraints) {
	p.Memory = c.Memory
	p.CPUShare = c.CPUShare
}

// Formation represents a collection of named processes and their configuration.
type Formation map[string]Process

// IsValid returns nil if all of the Processes are valid.
func (f Formation) IsValid() error {
	for n, p := range f {
		if err := p.IsValid(); err != nil {
			return fmt.Errorf("process %s is not valid: %v", n, err)
		}
	}

	return nil
}

// Merge merges in the existing quantity and constraints from the old Formation
// into this Formation.
func (f Formation) Merge(other Formation) Formation {
	new := make(Formation)

	for name, p := range f {
		if existing, found := other[name]; found {
			// If the existing Formation already had a process
			// configuration for this process type, copy over the
			// instance count.
			p.Quantity = existing.Quantity
			p.SetConstraints(existing.Constraints())
			p.Ulimits = existing.Ulimits
		} else {
			p.Quantity = DefaultQuantities[name]
			p.SetConstraints(DefaultConstraints)
		}

		new[name] = p
	}

	return new
}
