package empire

import (
	"encoding/json"
	"fmt"

	. "github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/constraints"
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

// Constraints aliases empire.Constraints type to implement the
// json.Unmarshaller interface.
type Constraints constraints.Constraints

func parseConstraints(con string) (*Constraints, error) {
	if con == "" {
		return nil, nil
	}

	if n, ok := NamedConstraints[con]; ok {
		c := Constraints(n)
		return &c, nil
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
		if c == Constraints(constraint) {
			return n
		}
	}

	if c.Nproc == 0 {
		return fmt.Sprintf("%d:%s", c.CPUShare, c.Memory)
	} else {
		return fmt.Sprintf("%d:%s:nproc=%d", c.CPUShare, c.Memory, c.Nproc)
	}
}
