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

	Formation *Formation `json:"-"`
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

// ProcessesRepository is an interface for creating and retreiving Processes.
type ProcessesRepository interface {
	// Create creates a new Process.
	Create(ProcessType, *Process) (ProcessType, *Process, error)

	// All returns the Processes that belong to a Formation.
	All(FormationID) (ProcessMap, error)
}

// NewProcessesRepository returns a new ProcessesRepository instance.
func NewProcessesRepository(db DB) (ProcessesRepository, error) {
	return &processesRepository{db}, nil
}

// dbProcess is the database representation of a Process.
type dbProcess struct {
	ID          string `db:"id"`
	FormationID string `db:"formation_id"`
	Type        string `db:"type"`
	Quantity    int64  `db:"quantity"`
	Command     string `db:"command"`
}

// processesRepository is an implementation of the AppsRepository interface backed by
// a DB.
type processesRepository struct {
	DB
}

func (r *processesRepository) Create(t ProcessType, process *Process) (ProcessType, *Process, error) {
	p := fromProcess(t, process)

	if err := r.DB.Insert(p); err != nil {
		return t, process, err
	}

	t, process = toProcess(p, process)
	return t, process, nil
}

// All a ProcessMap for a Formation.
func (r *processesRepository) All(id FormationID) (ProcessMap, error) {
	var ps []*dbProcess

	if err := r.DB.Select(`select * from processes where formation_id = $1`, string(id)); err != nil {
		return nil, err
	}

	pm := make(ProcessMap)

	for _, p := range ps {
		t, process := toProcess(p, nil)
		pm[t] = process
	}

	return pm, nil
}

func fromProcess(t ProcessType, process *Process) *dbProcess {
	return &dbProcess{
		FormationID: string(process.Formation.ID),
		Type:        string(t),
		Quantity:    int64(process.Quantity),
		Command:     string(process.Command),
	}
}

func toProcess(p *dbProcess, process *Process) (ProcessType, *Process) {
	if process == nil {
		process = &Process{}
	}

	process.Quantity = int(p.Quantity)
	process.Command = Command(p.Command)
	process.Formation = &Formation{ID: FormationID(p.FormationID)}

	return ProcessType(p.Type), process
}
