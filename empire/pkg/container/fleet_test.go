package container

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/coreos/fleet/schema"

	"text/template"
)

var testContainer = Container{
	Name: "acme-inc.v1.web.1",
	Env: map[string]string{
		"PORT":  "8080",
		"GOENV": "production",
	},
	Command: "acme-inc",
	Image: Image{
		Repo: "quay.io/ejholmes/acme-inc",
		Tag:  "ec238137726b58285f8951802aed0184f915323668487b4919aff2671c0f9a02",
	},
}

func TestDefaultTemplate(t *testing.T) {
	tmpl := DefaultTemplate

	buf := new(bytes.Buffer)
	if err := tmpl.Execute(buf, testContainer); err != nil {
		t.Fatal(err)
	}

	expected := `[Unit]
Description=acme-inc.v1.web.1
After=discovery.service

[Service]
TimeoutStartSec=30m
User=core
Restart=on-failure
KillMode=none

ExecStartPre=-/usr/bin/docker pull quay.io/ejholmes/acme-inc:ec238137726b58285f8951802aed0184f915323668487b4919aff2671c0f9a02
ExecStartPre=-/usr/bin/docker rm acme-inc.v1.web.1
ExecStart=/usr/bin/docker run --name acme-inc.v1.web.1 -e GOENV=production -e PORT=8080 --rm -h %H -P quay.io/ejholmes/acme-inc:ec238137726b58285f8951802aed0184f915323668487b4919aff2671c0f9a02 acme-inc
ExecStop=/usr/bin/docker stop acme-inc.v1.web.1

[X-Fleet]
MachineMetadata=role=empire_minion
`

	if got, want := buf.String(), expected; got != want {
		t.Fatal("Unit => %s\n====\n%s", got, want)
	}
}

func TestNewUnit(t *testing.T) {
	tests := []struct {
		tmpl      string
		container Container

		unit schema.Unit
	}{
		{
			simpleTemplate,
			testContainer,
			schema.Unit{
				Name: "acme-inc.v1.web.1.service",
				Options: []*schema.UnitOption{
					{
						Section: "Unit",
						Name:    "Description",
						Value:   "acme-inc.v1.web.1",
					},
					{
						Section: "Service",
						Name:    "ExecStart",
						Value:   "/usr/bin/docker run",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tmpl, err := template.New("").Parse(tt.tmpl)
		if err != nil {
			t.Fatal(err)
		}

		u, err := newUnit(tmpl, &tt.container)
		if err != nil {
			t.Fatal(err)
		}

		if got, want := u.Name, tt.unit.Name; got != want {
			t.Fatalf("Unit Name => %s; want %s", got, want)
		}

		if got, want := len(u.Options), len(tt.unit.Options); got != want {
			t.Fatalf("len(UnitOptions) => %d; want %d", got, want)
		}

		for i, opt := range tt.unit.Options {
			if got, want := u.Options[i], opt; !reflect.DeepEqual(got, want) {
				t.Fatalf("UnitOption[%d] => %v; want %v", i, got, want)
			}
		}
	}
}

var simpleTemplate = `
[Unit]
Description={{.Name}}

[Service]
ExecStart=/usr/bin/docker run
`
