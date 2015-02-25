package empire

// ProcessQuantityMap represents a map of process types to quantities
type ProcessQuantityMap map[ProcessType]int

// DefaultQuantities maps a process type to the default number of instances to
// run.
var DefaultQuantities = ProcessQuantityMap{
	"web": 1,
}

// ProcessType represents the type of a given process/command.
type ProcessType string

// Command represents the actual shell command that gets executed for a given
// ProcessType.
type Command string

// Process holds configuration information about a Process Type.
type Process struct {
	Quantity int     `json:"quantity"`
	Command  Command `json:"command"`
}

// CommandMap maps a process ProcessType to a Command.
type CommandMap map[ProcessType]Command

// ProcessMap maps a process ProcessType to a Process.
type ProcessMap map[ProcessType]*Process

// NewProcess returns a new Process instance.
func NewProcess(t ProcessType, cmd Command) *Process {
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
		p := NewProcess(t, cmd)

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
