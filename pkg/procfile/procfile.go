// Package procfile contains methods for parsing standard and extended
// Procfiles.
package procfile

import (
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

// Procfile is a Go representation of the Procfile format. The basic syntax of
// the standard Procfile format is described in https://devcenter.heroku.com/articles/procfile#declaring-process-types
type Procfile map[string]ProcessDefinition

// standardProcfile represents the standard Procfile format, without extended
// atrributes.
type standardProcfile map[string]string

func (p standardProcfile) Procfile() Procfile {
	procfile := make(Procfile)
	for k, command := range p {
		procfile[k] = ProcessDefinition{Command: command}
	}
	return procfile
}

// ProcessDefinition defines parameters about individual processes within the
// Procfile.
type ProcessDefinition struct {
	// The command that should be run when running this process definition.
	Command string

	// You can setup any health checks that should be performed against this
	// process to ensure that it's healthy.
	HealthChecks []HealthCheck
}

func (pd *ProcessDefinition) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var p yamlProcessDefinition
	if err := unmarshal(&p); err != nil {
		return err
	}

	var healthChecks []HealthCheck
	for _, v := range p.HealthChecks {
		t := v["type"]
		switch t {
		case "http":
			h, err := newHTTPHealthCheck(v)
			if err != nil {
				return err
			}
			healthChecks = append(healthChecks, h)
		case "tcp":
			healthChecks = append(healthChecks, TCPHealthCheck{})
		default:
			return fmt.Errorf("unknown health check type: %s", t)
		}
	}
	*pd = ProcessDefinition{
		Command:      p.Command,
		HealthChecks: healthChecks,
	}

	return nil
}

// an intermediated data structure used when yaml unmarshalling the process
// definition.
type yamlProcessDefinition struct {
	Command      string                   `yaml:"command"`
	HealthChecks []map[string]interface{} `yaml:"health_checks"`
}

// HealthCheck is an empty interface that represents a health check.
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
func Parse(r io.Reader) (Procfile, error) {
	raw, err := ioutil.ReadAll(r)
	if err != nil {
		return nil, err
	}

	var std standardProcfile
	if err := yaml.Unmarshal(raw, &std); err == nil {
		return std.Procfile(), nil
	}

	var procfile Procfile
	return procfile, yaml.Unmarshal(raw, &procfile)
}
