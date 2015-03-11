package container

import (
	"bytes"
	"fmt"
	"text/template"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/go-systemd/unit"
)

// FleetScheduler implements a scheduler that can schedule containers onto a
// comput cluster by using the fleet API.
type FleetScheduler struct {
	// UnitTemplate is a text.Template that will be used when generating
	// unit files to submit to fleet. This template will be Executed with a
	// Container as the data. The zero value will default to the parsed
	// DefaultTemplate.
	UnitTemplate *template.Template

	client client.API
}

func (s *FleetScheduler) Schedule(containers ...*Container) error {
	for _, container := range containers {
		if err := s.schedule(container); err != nil {
			return err
		}
	}

	return nil
}

func (s *FleetScheduler) Unschedule(names ...string) error {
	for _, name := range names {
		if err := s.unschedule(name); err != nil {
			return err
		}
	}

	return nil
}

func (s *FleetScheduler) schedule(container *Container) error {
	u, err := newUnit(s.unitTemplate(), container)
	if err != nil {
		return err
	}

	u.DesiredState = "launched"

	return s.client.CreateUnit(u)
}

func (s *FleetScheduler) unschedule(name string) error {
	return s.client.DestroyUnit(name)
}

func (s *FleetScheduler) unitTemplate() *template.Template {
	if s.UnitTemplate == nil {
		return DefaultTemplate
	}

	return s.UnitTemplate
}

// newUnit takes a systemd unit file template, and a container and:
//
//	* Parses the template, passing the container in as data.
//	* Deserializes the template into unit options.
//	* Generates a schema.Unit.
func newUnit(t *template.Template, container *Container) (*schema.Unit, error) {
	u := &schema.Unit{
		Name: fmt.Sprintf("%s.service", container.Name),
	}

	buf := new(bytes.Buffer)

	if err := t.Execute(buf, container); err != nil {
		return nil, err
	}

	opts, err := unit.Deserialize(buf)
	if err != nil {
		return nil, err
	}

	for _, opt := range opts {
		u.Options = append(u.Options, &schema.UnitOption{
			Section: opt.Section,
			Name:    opt.Name,
			Value:   opt.Value,
		})
	}

	return u, nil
}

// DefaultTemplate is the default systemd unit file template to use when
// scheduling units using fleet.
var DefaultTemplate = template.Must(template.New("").Parse(`[Unit]
Description={{.Name}}
After=discovery.service

[Service]
TimeoutStartSec=30m
User=core
Restart=on-failure
KillMode=none

ExecStartPre=-/usr/bin/docker pull {{.Image}}
ExecStartPre=-/usr/bin/docker rm {{.Name}}
ExecStart=/usr/bin/docker run --name {{.Name}}{{range $key, $val := .Env}} -e {{$key}}={{$val}}{{end}} --rm -h %H -P {{.Image}} {{.Command}}
ExecStop=/usr/bin/docker stop {{.Name}}

[X-Fleet]
MachineMetadata=role=empire_minion
`))
