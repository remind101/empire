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

// Scan implements the sql.Scanner interface.
func (p *ProcessType) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*p = ProcessType(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (p ProcessType) Value() (driver.Value, error) {
	return driver.Value(string(p)), nil
}

// Command represents the actual shell command that gets executed for a given
// ProcessType.
type Command string

// Scan implements the sql.Scanner interface.
func (c *Command) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		*c = Command(src)
	}

	return nil
}

// Value implements the driver.Value interface.
func (c Command) Value() (driver.Value, error) {
	return driver.Value(string(c)), nil
}

// Process holds configuration information about a Process Type.
type Process struct {
	ID       string      `json:"id" db:"id"`
	Type     ProcessType `json:"type" db:"type"`
	Quantity int         `json:"quantity" db:"quantity"`
	Command  Command     `json:"command" db:"command"`

	ReleaseID string `json:"-" db:"release_id"`
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
		Type:     t,
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

type ProcessesCreator interface {
	ProcessesCreate(*Process) (*Process, error)
}

type ProcessesUpdater interface {
	ProcessesUpdate(*Process) (int64, error)
}

type ProcessesFinder interface {
	ProcessesAll(*Release) (Formation, error)
}

type ProcessesService interface {
	ProcessesCreator
	ProcessesUpdater
	ProcessesFinder
}

// processesService is an implementation of the AppsRepository interface backed by
// a DB.
type processesService struct {
	*db
}

func (s *processesService) ProcessesCreate(process *Process) (*Process, error) {
	return processesCreate(s.db, process)
}

func (s *processesService) ProcessesUpdate(process *Process) (int64, error) {
	return processesUpdate(s.db, process)
}

func (s *processesService) ProcessesAll(release *Release) (Formation, error) {
	return processesAll(s.db, release)
}

// ProcessesCreate inserts a process into the database.
func processesCreate(db *db, process *Process) (*Process, error) {
	return process, db.Insert(process)
}

// ProcessesUpdate updates an existing process into the database.
func processesUpdate(db *db, process *Process) (int64, error) {
	return db.Update(process)
}

// ProcessesAll returns all Processes for a Release as a Formation.
func processesAll(db *db, release *Release) (Formation, error) {
	var ps []*Process

	if err := db.Select(&ps, `select * from processes where release_id = $1`, string(release.ID)); err != nil {
		return nil, err
	}

	f := make(Formation)

	for _, p := range ps {
		f[p.Type] = p
	}

	return f, nil
}
