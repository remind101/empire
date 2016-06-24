// Package procfile provides methods for parsing standard and extended
// Procfiles.
package procfile

import (
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Procfile is a Go representation of process configuration.
type Procfile interface {
	version() string
}

// ExtendedProcfile represents the extended Procfile format.
type ExtendedProcfile map[string]Process

func (e ExtendedProcfile) version() string {
	return "extended"
}

type Process struct {
	Command interface{} `yaml:"command"`
}

// StandardProcfile represents a standard Procfile.
type StandardProcfile map[string]string

func (p StandardProcfile) version() string {
	return "standard"
}

// Marshal marshals the Procfile to yaml format.
func Marshal(p Procfile) ([]byte, error) {
	return yaml.Marshal(p)
}

// Parse parses the Procfile by reading from r.
func Parse(r io.Reader) (Procfile, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	return ParseProcfile(raw)
}

// ParseProcfile takes a byte slice representing a YAML Procfile and parses it
// into a Procfile.
func ParseProcfile(b []byte) (Procfile, error) {
	p, err := parseStandardProcfile(b)
	if err != nil {
		p, err = parseExtendedProcfile(b)
	}
	return p, err
}

func parseExtendedProcfile(b []byte) (Procfile, error) {
	y := make(ExtendedProcfile)
	err := yaml.Unmarshal(b, &y)
	return y, err
}

func parseStandardProcfile(b []byte) (Procfile, error) {
	y := make(StandardProcfile)
	err := yaml.Unmarshal(b, &y)
	return y, err
}
