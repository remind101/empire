package empire

import (
	"database/sql"
	"database/sql/driver"

	"github.com/lib/pq/hstore"
)

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

	Release *Release `json:"-"`
}

// CommandMap maps a process ProcessType to a Command.
type CommandMap map[ProcessType]Command

// Scan implements the sql.Scanner interface.
func (cm *CommandMap) Scan(src interface{}) error {
	h := hstore.Hstore{}
	if err := h.Scan(src); err != nil {
		return err
	}

	m := make(CommandMap)

	for k, v := range h.Map {
		m[ProcessType(k)] = Command(v.String)
	}

	*cm = m

	return nil
}

// Value implements the driver.Value interface.
func (cm CommandMap) Value() (driver.Value, error) {
	m := make(map[string]sql.NullString)

	for k, v := range cm {
		m[string(k)] = sql.NullString{
			Valid:  true,
			String: string(v),
		}
	}

	h := hstore.Hstore{
		Map: m,
	}

	return h.Value()
}

// Formation maps a process ProcessType to a Process.
type Formation map[ProcessType]*Process

// NewProcess returns a new Process instance.
func NewProcess(t ProcessType, cmd Command) *Process {
	return &Process{
		Quantity: DefaultQuantities[t],
		Command:  cmd,
	}
}

// NewFormation creates a new Formation based on an existing Formation and
// the available processes from a CommandMap.
func NewFormation(f Formation, cm CommandMap) Formation {
	processes := make(Formation)

	// Iterate through all of the available process types in the CommandMap.
	for t, cmd := range cm {
		p := NewProcess(t, cmd)

		if existing, found := f[t]; found {
			// If the existing Formation already had a process
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
	All(ReleaseID) (Formation, error)
}

// NewProcessesRepository returns a new ProcessesRepository instance.
func NewProcessesRepository(db DB) (ProcessesRepository, error) {
	return &processesRepository{db}, nil
}

// dbProcess is the database representation of a Process.
type dbProcess struct {
	ID        string `db:"id"`
	ReleaseID string `db:"release_id"`
	Type      string `db:"type"`
	Quantity  int64  `db:"quantity"`
	Command   string `db:"command"`
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

// All a Formation for a Formation.
func (r *processesRepository) All(id ReleaseID) (Formation, error) {
	var ps []*dbProcess

	if err := r.DB.Select(`select * from processes where release_id = $1`, string(id)); err != nil {
		return nil, err
	}

	f := make(Formation)

	for _, p := range ps {
		t, process := toProcess(p, nil)
		f[t] = process
	}

	return f, nil
}

func fromProcess(t ProcessType, process *Process) *dbProcess {
	return &dbProcess{
		ReleaseID: string(process.Release.ID),
		Type:      string(t),
		Quantity:  int64(process.Quantity),
		Command:   string(process.Command),
	}
}

func toProcess(p *dbProcess, process *Process) (ProcessType, *Process) {
	if process == nil {
		process = &Process{}
	}

	process.Quantity = int(p.Quantity)
	process.Command = Command(p.Command)
	process.Release = &Release{ID: ReleaseID(p.ReleaseID)}

	return ProcessType(p.Type), process
}
