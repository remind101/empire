package empire

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/jinzhu/gorm"
	"github.com/lib/pq/hstore"
	. "github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/constraints"
	"github.com/remind101/empire/pkg/service"
	"golang.org/x/net/context"
)

var (
	Constraints1X = Constraints{constraints.CPUShare(256), constraints.Memory(512 * MB)}
	Constraints2X = Constraints{constraints.CPUShare(512), constraints.Memory(1 * GB)}
	ConstraintsPX = Constraints{constraints.CPUShare(1024), constraints.Memory(6 * GB)}

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
	ID       string
	Type     ProcessType
	Quantity int
	Command  Command
	Port     int `sql:"-"`
	Constraints

	ReleaseID string
	Release   *Release
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

	return fmt.Sprintf("%d:%s", c.CPUShare, c.Memory)
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

// ProcessesQuery is a Scope implementation for common things to filter
// processes by.
type ProcessesQuery struct {
	// If provided, finds only processes belonging to the given release.
	Release *Release
}

// Scope implements the Scope interface.
func (q ProcessesQuery) Scope(db *gorm.DB) *gorm.DB {
	var scope ComposedScope

	if q.Release != nil {
		scope = append(scope, FieldEquals("release_id", q.Release.ID))
	}

	return scope.Scope(db)
}

// Processes returns all processes matching the scope.
func (s *store) Processes(scope Scope) ([]*Process, error) {
	var processes []*Process
	return processes, s.Find(scope, &processes)
}

// Formation returns a Formation for the processes matching the scope.
func (s *store) Formation(scope Scope) (Formation, error) {
	p, err := s.Processes(scope)
	if err != nil {
		return nil, err
	}
	return newFormation(p), nil
}

// ProcessesCreate persists the process.
func (s *store) ProcessesCreate(process *Process) (*Process, error) {
	return processesCreate(s.db, process)
}

// ProcessesUpdate updates the process.
func (s *store) ProcessesUpdate(process *Process) error {
	return processesUpdate(s.db, process)
}

// ProcessesCreate inserts a process into the database.
func processesCreate(db *gorm.DB, process *Process) (*Process, error) {
	return process, db.Create(process).Error
}

// ProcessesUpdate updates an existing process into the database.
func processesUpdate(db *gorm.DB, process *Process) error {
	return db.Save(process).Error
}

// ProcessState represents the state of a Process.
type ProcessState struct {
	Name        string
	Type        string
	Command     string
	State       string
	UpdatedAt   time.Time
	Constraints Constraints
}

type processStatesService struct {
	manager service.Manager
}

func (s *processStatesService) JobStatesByApp(ctx context.Context, app *App) ([]*ProcessState, error) {
	var states []*ProcessState

	instances, err := s.manager.Instances(ctx, app.ID)
	if err != nil {
		return states, err
	}

	for _, i := range instances {
		states = append(states, processStateFromInstance(i))
	}

	return states, nil
}

// processStateFromInstance converts a service.Instance into a ProcessState.
// It pulls some of its data from empire specific environment variables if they have been set.
// Once ECS supports this data natively, we can stop doing this.
func processStateFromInstance(i *service.Instance) *ProcessState {
	createdAt := i.UpdatedAt
	if t, err := time.Parse(time.RFC3339, i.Process.Env["EMPIRE_CREATED_AT"]); err == nil {
		createdAt = t
	}

	version := i.Process.Env["EMPIRE_RELEASE"]
	if version == "" {
		version = "v0"
	}

	return &ProcessState{
		Name:    fmt.Sprintf("%s.%s.%s", version, i.Process.Type, i.ID),
		Type:    string(i.Process.Type),
		Command: i.Process.Command,
		Constraints: Constraints{
			CPUShare: constraints.CPUShare(i.Process.CPUShares),
			Memory:   constraints.Memory(i.Process.MemoryLimit),
		},
		State:     i.State,
		UpdatedAt: createdAt, // This is the best data we have, until ECS gives us UpdatedAt
	}
}
