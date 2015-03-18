package container

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"text/template"

	"github.com/coreos/fleet/client"
	"github.com/coreos/fleet/schema"
	"github.com/coreos/go-systemd/unit"
)

// Ensure that FleetScheduler conforms to the Scheduler interface.
var _ Scheduler = &FleetScheduler{}

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

// NewFleetScheduler returns a new FleetScheduler instance with the fleet client
// pointed at api.
func NewFleetScheduler(api *url.URL) (*FleetScheduler, error) {
	c, err := client.NewHTTPClient(http.DefaultClient, *api)
	if err != nil {
		return nil, err
	}

	return &FleetScheduler{
		client: c,
	}, nil
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

func (s *FleetScheduler) ContainerStates() ([]*ContainerState, error) {
	states, err := s.client.UnitStates()
	if err != nil {
		return nil, err
	}

	cs := make([]*ContainerState, len(states))
	for i, state := range states {
		cs[i] = &ContainerState{
			Container: &Container{
				Name: containerNameFromUnitName(state.Name),
			},
			MachineID: state.MachineID,
			State:     state.SystemdActiveState,
		}
	}

	return cs, nil
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
	return s.client.DestroyUnit(unitNameFromContainerName(name))
}

func (s *FleetScheduler) unitTemplate() *template.Template {
	if s.UnitTemplate == nil {
		return DefaultTemplate
	}

	return s.UnitTemplate
}

func unitNameFromContainerName(name string) string {
	return fmt.Sprintf("%s.service", name)
}

func containerNameFromUnitName(name string) string {
	return strings.TrimSuffix(name, ".service")
}

// newUnit takes a systemd unit file template, and a container and:
//
//	* Parses the template, passing the container in as data.
//	* Deserializes the template into unit options.
//	* Generates a schema.Unit.
func newUnit(t *template.Template, container *Container) (*schema.Unit, error) {
	u := &schema.Unit{
		Name: unitNameFromContainerName(container.Name),
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

ExecStartPre=/bin/sh -c "> /tmp/{{.Name}}.env"
{{$name := .Name}}
{{range $key, $val := .Env}}
ExecStartPre=/bin/sh -c "echo {{$key}}={{$val}} >> /tmp/{{$name}}.env"
{{end}}

ExecStartPre=-/usr/bin/docker pull {{.Image}}
ExecStartPre=-/usr/bin/docker rm {{.Name}}
ExecStart=/usr/bin/docker run --name {{.Name}} --env-file /tmp/{{.Name}}.env -e PORT=80 -h %H -p 80 {{.Image}} sh -c '{{.Command}}'
ExecStop=/usr/bin/docker stop {{.Name}}

[X-Fleet]
MachineMetadata=role=empire_minion
`))
