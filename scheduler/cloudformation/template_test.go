package cloudformation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/pkg/troposphere"
	"github.com/remind101/empire/scheduler"
	"github.com/stretchr/testify/assert"
)

func TestEmpireTemplate(t *testing.T) {
	tests := []struct {
		file string
		app  *scheduler.App
	}{
		{
			"basic.json",
			&scheduler.App{
				ID:      "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					// These should get re-sorted in
					// alphabetical order.
					"C": "foo",
					"A": "foobar",
					"B": "bar",
				},
				Processes: []*scheduler.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &scheduler.Exposure{
							Ports: []scheduler.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &scheduler.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						MemoryLimit: 128 * bytesize.MB,
						CPUShares:   256,
						Instances:   1,
						Nproc:       256,
					},
					{
						Type:    "worker",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/worker"},
						Labels: map[string]string{
							"empire.app.process": "worker",
						},
						Env: map[string]string{
							"FOO": "BAR",
						},
					},
				},
			},
		},

		{
			"basic-alb.json",
			&scheduler.App{
				ID:      "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					// These should get re-sorted in
					// alphabetical order.
					"C": "foo",
					"A": "foobar",
					"B": "bar",

					"LOAD_BALANCER_TYPE": "alb",
				},
				Processes: []*scheduler.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Exposure: &scheduler.Exposure{
							Ports: []scheduler.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &scheduler.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Env: map[string]string{
							"PORT": "8080",
						},
						MemoryLimit: 128 * bytesize.MB,
						CPUShares:   256,
						Instances:   1,
						Nproc:       256,
					},
					{
						Type:    "worker",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/worker"},
						Labels: map[string]string{
							"empire.app.process": "worker",
						},
						Env: map[string]string{
							"FOO": "BAR",
						},
					},
				},
			},
		},

		{
			"https.json",
			&scheduler.App{
				ID:      "1234",
				Release: "v1",
				Name:    "acme-inc",
				Processes: []*scheduler.Process{
					{
						Type:    "web",
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &scheduler.Exposure{
							Ports: []scheduler.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &scheduler.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &scheduler.HTTPS{
										Cert: "arn:aws:iam::012345678901:server-certificate/AcmeIncDotCom",
									},
								},
							},
						},
					},
					{
						Type:    "api",
						Command: []string{"./bin/api"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &scheduler.Exposure{
							Ports: []scheduler.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &scheduler.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &scheduler.HTTPS{
										Cert: "AcmeIncDotCom", // Simple cert format.
									},
								},
							},
						},
					},
				},
			},
		},

		{
			"https-alb.json",
			&scheduler.App{
				ID:      "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					"LOAD_BALANCER_TYPE": "alb",
				},
				Processes: []*scheduler.Process{
					{
						Type:    "web",
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &scheduler.Exposure{
							Ports: []scheduler.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &scheduler.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &scheduler.HTTPS{
										Cert: "arn:aws:iam::012345678901:server-certificate/AcmeIncDotCom",
									},
								},
							},
						},
					},
					{
						Type:    "api",
						Command: []string{"./bin/api"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &scheduler.Exposure{
							Ports: []scheduler.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &scheduler.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &scheduler.HTTPS{
										Cert: "AcmeIncDotCom", // Simple cert format
									},
								},
							},
						},
					},
				},
			},
		},

		{
			"custom.json",
			&scheduler.App{
				ID:      "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					"ECS_TASK_DEFINITION": "custom",
				},
				Processes: []*scheduler.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"B":    "foo",
							"A":    "foo",
							"FOO":  "bar",
							"PORT": "8080",
						},
						Exposure: &scheduler.Exposure{
							Ports: []scheduler.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &scheduler.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						MemoryLimit: 128 * bytesize.MB,
						CPUShares:   256,
						Instances:   1,
						Nproc:       256,
					},
					{
						Type:      "vacuum",
						Image:     image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command:   []string{"./bin/vacuum"},
						Schedule:  scheduler.CRONSchedule("* * * * *"),
						Instances: 1,
						Labels: map[string]string{
							"empire.app.process": "vacuum",
						},
						MemoryLimit: 128 * bytesize.MB,
						CPUShares:   256,
						Nproc:       256,
					},
				},
			},
		},

		{
			"cron.json",
			&scheduler.App{
				ID:      "1234",
				Release: "v1",
				Name:    "acme-inc",
				Processes: []*scheduler.Process{
					{
						Type:      "send-emails",
						Image:     image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command:   []string{"./bin/send-emails"},
						Schedule:  scheduler.CRONSchedule("* * * * *"),
						Instances: 1,
						Labels: map[string]string{
							"empire.app.process": "send-emails",
						},
						MemoryLimit: 128 * bytesize.MB,
						CPUShares:   256,
						Nproc:       256,
					},
					{
						Type:      "vacuum",
						Image:     image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command:   []string{"./bin/vacuum"},
						Schedule:  scheduler.CRONSchedule("* * * * *"),
						Instances: 0,
						Labels: map[string]string{
							"empire.app.process": "vacuum",
						},
						MemoryLimit: 128 * bytesize.MB,
						CPUShares:   256,
						Nproc:       256,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tmpl := newTemplate()
		tmpl.NoCompress = true
		buf := new(bytes.Buffer)

		filename := fmt.Sprintf("templates/%s", tt.file)
		err := tmpl.Execute(buf, tt.app)
		assert.NoError(t, err)

		expected, err := ioutil.ReadFile(filename)
		assert.NoError(t, err)

		assert.Equal(t, string(expected), buf.String())
		ioutil.WriteFile(filename, buf.Bytes(), 0660)
	}
}

func TestEmpireTemplate_Large(t *testing.T) {
	labels := make(map[string]string)
	env := make(map[string]string)
	for i := 0; i < 100; i++ {
		env[fmt.Sprintf("ENV_VAR_%d", i)] = fmt.Sprintf("value%d", i)
	}
	app := &scheduler.App{
		ID:      "",
		Release: "v1",
		Name:    "bigappwithlotsofprocesses",
		Env:     env,
		Labels:  labels,
	}

	for i := 0; i < 60; i++ {
		app.Processes = append(app.Processes, &scheduler.Process{
			Type:    fmt.Sprintf("%d", i),
			Command: []string{"./bin/web"},
		})
	}

	tmpl := newTemplate()
	buf := new(bytes.Buffer)

	err := tmpl.Execute(buf, app)
	t.Logf("Template size: %d bytes", buf.Len())
	assert.NoError(t, err)
	assert.Condition(t, func() bool {
		return buf.Len() < MaxTemplateSize
	}, fmt.Sprintf("template must be smaller than %d, was %d", MaxTemplateSize, buf.Len()))
}

func TestScheduleExpression(t *testing.T) {
	tests := []struct {
		schedule   scheduler.Schedule
		expression string
	}{
		{scheduler.CRONSchedule("0 12 * * ? *"), "cron(0 12 * * ? *)"},
		{5 * time.Minute, "rate(5 minutes)"},
		{1 * time.Minute, "rate(1 minute)"},
		{24 * time.Hour, "rate(1440 minutes)"},
	}

	for _, tt := range tests {
		expression := scheduleExpression(tt.schedule)
		assert.Equal(t, tt.expression, expression)
	}
}

func newTemplate() *EmpireTemplate {
	return &EmpireTemplate{
		Cluster:                 "cluster",
		ServiceRole:             "ecsServiceRole",
		InternalSecurityGroupID: "sg-e7387381",
		ExternalSecurityGroupID: "sg-1938737f",
		InternalSubnetIDs:       []string{"subnet-bb01c4cd", "subnet-c85f4091"},
		ExternalSubnetIDs:       []string{"subnet-ca96f4cd", "subnet-a13b909c"},
		CustomResourcesTopic:    "sns topic arn",
		HostedZone: &route53.HostedZone{
			Id:   aws.String("Z3DG6IL3SJCGPX"),
			Name: aws.String("empire"),
		},
		ExtraOutputs: map[string]troposphere.Output{
			"EmpireVersion": troposphere.Output{
				Value: "x.x.x",
			},
		},
	}
}
