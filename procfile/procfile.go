// Package procfile provides methods for parsing standard and extended
// Procfiles.
package procfile

import (
	"fmt"
	"io"
	"io/ioutil"
	"strconv"
	"strings"

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
	Command   interface{} `yaml:"command"`
	Cron      *string     `yaml:"cron,omitempty"`
	NoService bool        `yaml:"noservice,omitempty"`
	Ports     []Port      `yaml:"ports,omitempty"`
}

// Port represents a port mapping.
type Port struct {
	Host      int
	Container int
	Protocol  string
}

// ParsePort parses a string into a Port.
func ParsePort(s string) (p Port, err error) {
	if strings.Contains(s, ":") {
		return portFromHostContainer(s)
	}

	var port int
	port, err = toPort(s)
	if err != nil {
		return
	}
	p.Host = port
	p.Container = port
	return
}

// Parses a `hostport:containerport` string into a Port.
func portFromHostContainer(hostContainer string) (p Port, err error) {
	var host, container int
	parts := strings.SplitN(hostContainer, ":", 2)
	host, err = toPort(parts[0])
	if err != nil {
		return
	}
	container, err = toPort(parts[1])
	if err != nil {
		return
	}

	p.Host = host
	p.Container = container
	return
}

func toPort(s string) (port int, err error) {
	port, err = strconv.Atoi(s)
	if err != nil {
		err = fmt.Errorf("error converting %s to port from: %v", s, err)
	}
	return
}

func (p *Port) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	err := unmarshal(&s)
	if err == nil {
		port, err := ParsePort(s)
		if err != nil {
			return err
		}

		*p = port
		return nil
	}

	m := make(map[interface{}]map[interface{}]interface{})
	err = unmarshal(&m)
	if err == nil {
		if len(m) > 1 {
			return fmt.Errorf("invalid port format")
		}

		for k, v := range m {
			port, err := ParsePort(k.(string))
			if err != nil {
				return err
			}
			*p = port
			p.Protocol = v[interface{}("protocol")].(string)
			return nil
		}
	}

	return err
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
