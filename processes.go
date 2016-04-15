package empire

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq/hstore"
	shellwords "github.com/mattn/go-shellwords"
	. "github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/constraints"
	"github.com/remind101/empire/procfile"
)

var (
	Constraints1X = Constraints{constraints.CPUShare(256), constraints.Memory(512 * MB), constraints.Nproc(256)}
	Constraints2X = Constraints{constraints.CPUShare(512), constraints.Memory(1 * GB), constraints.Nproc(512)}
	ConstraintsPX = Constraints{constraints.CPUShare(1024), constraints.Memory(6 * GB), 0}

	// NamedConstraints maps a heroku dynos size to a Constraints.
	NamedConstraints = map[string]Constraints{
		"1X": Constraints1X,
		"2X": Constraints2X,
		"PX": ConstraintsPX,
	}

	// DefaultConstraints defaults to 1X process size.
	DefaultConstraints = Constraints1X
)

// ProcessQuantityMap represents a map of process types to quantities.
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
type Command []string

// ParseCommand parses a string into a Command, taking quotes and other shell
// words into account.
func ParseCommand(command string) (Command, error) {
	return shellwords.Parse(command)
}

// Scan implements the sql.Scanner interface.
func (c *Command) Scan(src interface{}) error {
	if src, ok := src.([]byte); ok {
		command, err := ParseCommand(string(src))
		if err != nil {
			return err
		}
		*c = command
	}

	return nil
}

// Value implements the driver.Value interface.
func (c Command) Value() (driver.Value, error) {
	// TODO(ejholmes): We really should be storing this as a postgres array,
	// because stringifying it can cause information to be lost.
	//
	// For example, if we have the command:
	//
	//	Command{"echo", "hello world"}
	//
	// Then stringify it:
	//
	//	"echo hello world"
	//
	// Then parse it again:
	//
	//	Command{"echo", "hello", "world"}
	return driver.Value(c.String()), nil
}

// String returns the string reprsentation of the command.
func (c Command) String() string {
	return strings.Join([]string(c), " ")
}

// Process holds configuration information about a Process Type.
type Process struct {
	ReleaseID string
	ID        string
	Type      ProcessType
	Quantity  int
	Command   Command
	Constraints
}

// NewProcess returns a new Process instance.
func NewProcess(t ProcessType, cmd Command) *Process {
	return &Process{
		Type:        t,
		Quantity:    DefaultQuantities[t],
		Command:     cmd,
		Constraints: DefaultConstraints,
	}
}

// CommandMap maps a process ProcessType to a Command.
type CommandMap map[ProcessType]Command

func commandMapFromProcfile(p procfile.Procfile) CommandMap {
	cm := make(CommandMap)
	for n, c := range p {
		cm[ProcessType(n)] = Command(c)
	}
	return cm
}

// Scan implements the sql.Scanner interface.
func (cm *CommandMap) Scan(src interface{}) error {
	h := hstore.Hstore{}
	if err := h.Scan(src); err != nil {
		return err
	}

	m := make(CommandMap)

	for k, v := range h.Map {
		command, err := ParseCommand(v.String)
		if err != nil {
			return err
		}

		m[ProcessType(k)] = command
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
			String: v.String(),
		}
	}

	h := hstore.Hstore{
		Map: m,
	}

	return h.Value()
}

// Constraints aliases constraints.Constraints to implement the
// sql.Scanner interface.
type Constraints constraints.Constraints

func parseConstraints(con string) (*Constraints, error) {
	if con == "" {
		return nil, nil
	}

	if n, ok := NamedConstraints[con]; ok {
		return &n, nil
	}

	c, err := constraints.Parse(con)
	if err != nil {
		return nil, err
	}

	r := Constraints(c)
	return &r, nil
}

func (c *Constraints) UnmarshalJSON(b []byte) error {
	var s string

	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	cc, err := parseConstraints(s)
	if err != nil {
		return err
	}

	if cc != nil {
		*c = *cc
	}

	return nil
}

func (c Constraints) String() string {
	for n, constraint := range NamedConstraints {
		if c == constraint {
			return n
		}
	}

	if c.Nproc == 0 {
		return fmt.Sprintf("%d:%s", c.CPUShare, c.Memory)
	} else {
		return fmt.Sprintf("%d:%s:nproc=%d", c.CPUShare, c.Memory, c.Nproc)
	}
}

// Formation maps a process ProcessType to a Process.
type Formation map[ProcessType]*Process

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
			p.Constraints = existing.Constraints
		}

		processes[t] = p
	}

	return processes
}

// newFormation takes a slice of processes and returns a Formation.
func newFormation(p []*Process) Formation {
	f := make(Formation)

	for _, pp := range p {
		f[pp.Type] = pp
	}

	return f
}

// Processes takes a Formation and returns a slice of the processes.
func (f Formation) Processes() []*Process {
	var processes []*Process

	for _, p := range f {
		processes = append(processes, p)
	}

	return processes
}

// processesUpdate updates an existing process into the database.
func processesUpdate(db *gorm.DB, process *Process) error {
	return db.Save(process).Error
}
