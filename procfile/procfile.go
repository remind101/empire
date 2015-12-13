// Package procfile contains methods for parsing standard and extended
// Procfiles.
package procfile

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// ErrBadProcfile is returned by Parse when the given procfile is invalid.
var ErrBadProcfile = errors.New("procfile: unknown Procfile format")

// Procfile represents the decoded Go representation of a Procfile.
type Procfile map[string]ProcessDefinition

// StandardProcfile represents the standard Procfile format, without extended
// atrributes.
type StandardProcfile map[string]string

func (p StandardProcfile) Procfile() (procfile Procfile, err error) {
	procfile = make(Procfile)

	for k, command := range p {
		procfile[k] = ProcessDefinition{
			Name:    k,
			Command: command,
		}
	}

	return
}

// ExtendedProcfile represents the extended Procfile format. This format is also
// compatible with the docker-compose.yml format.
type ExtendedProcfile map[string]struct {
	Command      string                   `yaml:"command"`
	HealthChecks []map[string]interface{} `yaml:"health_checks"`
}

func (p ExtendedProcfile) Procfile() (procfile Procfile, err error) {
	procfile = make(Procfile)

	for k, pd := range p {
		var healthChecks []HealthCheck
		for _, v := range pd.HealthChecks {
			t := v["type"]
			switch t {
			case "http":
				var h HealthCheck
				h, err = newHTTPHealthCheck(v)
				if err != nil {
					return
				}
				healthChecks = append(healthChecks, h)
			case "tcp":
				healthChecks = append(healthChecks, TCPHealthCheck{})
			default:
				err = fmt.Errorf("unknown health check type: %s", t)
				return
			}
		}

		procfile[k] = ProcessDefinition{
			Name:         k,
			Command:      pd.Command,
			HealthChecks: healthChecks,
		}
	}
	return
}

// ProcessDefinition defines parameters about individual processes within the
// Procfile.
type ProcessDefinition struct {
	// The name of the process.
	Name string

	// The command that should be run when running this process definition.
	Command string

	// You can setup any health checks that should be performed against this
	// process to ensure that it's healthy.
	HealthChecks []HealthCheck
}

// HealthCheck is an interface that represents a health check.
type HealthCheck interface {
	Type() string
}

// HTTPHealthCheck represents a health check that will perform an http request
// to a specified port.
type HTTPHealthCheck struct {
	Path     string
	Timeout  int
	Interval int
}

func newHTTPHealthCheck(v map[string]interface{}) (HTTPHealthCheck, error) {
	return HTTPHealthCheck{
		Path:     v["path"].(string),
		Timeout:  v["timeout"].(int),
		Interval: v["interval"].(int),
	}, nil
}

func (hc HTTPHealthCheck) Type() string {
	return "http"
}

// TCPHealthCheck represents a health check that will connect to the container
// on a given port.
type TCPHealthCheck struct{}

func (hc TCPHealthCheck) Type() string {
	return "tcp"
}

// Parses parses the Procfile read from r and returns the Go representation.
//
// It attempts to parse the Procfile using the following formats:
//
// 1. Try to parse the Standard procfile format.
// 2. Try to parse the Extended procfile format.
func Parse(r io.Reader) (Procfile, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var std StandardProcfile
	if err := yaml.Unmarshal(raw, &std); err == nil {
		return std.Procfile()
	}

	var extd ExtendedProcfile
	if err := yaml.Unmarshal(raw, &extd); err == nil {
		return extd.Procfile()
	}

	return nil, ErrBadProcfile
}
