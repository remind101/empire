package processes

// DefaultQuantities maps a process type to the default number of instances to
// run.
var DefaultQuantities = map[Type]int{
	"web": 1,
}

// Type represents the type of a given process/command.
type Type string

// Command represents the actual shell command that gets executed for a given
// ProcessType.
type Command string

// Process holds configuration information about a Process Type.
type Process struct {
	Quantity int     `json:"quantity"`
	Command  Command `json:"command"`
}

// CommandMap maps a process Type to a Command.
type CommandMap map[Type]Command

// ProcessMap maps a process Type to a Process.
type ProcessMap map[Type]*Process

// New returns a new Process instance.
func New(t Type, cmd Command) *Process {
	return &Process{
		Quantity: DefaultQuantities[t],
		Command:  cmd,
	}
}

// NewProcessMap creates a new ProcessMap based on an existing ProcessMap and
// the available processes from a CommandMap.
func NewProcessMap(pm ProcessMap, cm CommandMap) ProcessMap {
	processes := make(ProcessMap)

	// Iterate through all of the available process types in the CommandMap.
	for t, cmd := range cm {
		p := New(t, cmd)

		if existing, found := pm[t]; found {
			// If the existing ProcessMap already had a process
			// configuration for this process type, copy over the
			// instance count.
			p.Quantity = existing.Quantity
		}

		processes[t] = p
	}

	return processes
}
