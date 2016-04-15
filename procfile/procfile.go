package procfile

import (
	"github.com/mattn/go-shellwords"
	"gopkg.in/yaml.v2"
)

// Procfile is a Go representation of a Procfile, which maps a named process to
// a command to run.
type Procfile map[string][]string

// yamlProcfile is a struct that we can yaml.Unmarshal into.
type yamlProcfile map[string]string

// ParseProcfile takes a byte slice representing a YAML Procfile and parses it
// into a Procfile.
func ParseProcfile(b []byte) (Procfile, error) {
	y := make(yamlProcfile)

	if err := yaml.Unmarshal(b, &y); err != nil {
		return nil, err
	}

	p := make(Procfile)
	for process, command := range y {
		args, err := shellwords.Parse(command)
		if err != nil {
			return nil, err
		}
		p[process] = args
	}

	return p, nil
}
