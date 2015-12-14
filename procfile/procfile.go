package procfile

import "gopkg.in/yaml.v2"

// Procfile is a Go representation of a Procfile, which maps a named process to
// a command to run.
//
// TODO: This would be better represented as a map[string][]string.
type Procfile map[string]string

// ParseProcfile takes a byte slice representing a YAML Procfile and parses it
// into a Procfile.
func ParseProcfile(b []byte) (Procfile, error) {
	p := make(Procfile)

	if err := yaml.Unmarshal(b, &p); err != nil {
		return p, err
	}

	return p, nil
}
