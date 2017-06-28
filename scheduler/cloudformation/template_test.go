package cloudformation

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/image"
	"github.com/remind101/empire/pkg/troposphere"
	"github.com/remind101/empire/twelvefactor"
	"github.com/stretchr/testify/assert"
)

func TestEmpireTemplate(t *testing.T) {
	tests := []struct {
		newTemplate func() *EmpireTemplate
		file        string
		app         *twelvefactor.Manifest
	}{
		{
			newTemplate,
			"basic.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					// These should get re-sorted in
					// alphabetical order.
					"C": "foo",
					"A": "foobar",
					"B": "bar",
				},
				Processes: []*twelvefactor.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
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
			newTemplate,
			"basic-alb.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
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
				Processes: []*twelvefactor.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Env: map[string]string{
							"PORT": "8080",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
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
			newTemplate,
			"https.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Processes: []*twelvefactor.Process{
					{
						Type:    "web",
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &twelvefactor.HTTPS{
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
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &twelvefactor.HTTPS{
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
			newTemplate,
			"https-alb.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Processes: []*twelvefactor.Process{
					{
						Type:    "web",
						Command: []string{"./bin/web"},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Env: map[string]string{
							"PORT":               "8080",
							"LOAD_BALANCER_TYPE": "alb",
						},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &twelvefactor.HTTPS{
										Cert: "arn:aws:iam::012345678901:server-certificate/AcmeIncDotCom",
									},
								},
							},
						},
					},
					{
						Type:    "api",
						Command: []string{"./bin/api"},
						Labels: map[string]string{
							"empire.app.process": "api",
						},
						Env: map[string]string{
							"PORT": "8080",
							"EMPIRE_X_LOAD_BALANCER_TYPE": "alb",
						},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
								{
									Host:      443,
									Container: 8080,
									Protocol: &twelvefactor.HTTPS{
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
			newTemplate,
			"custom.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					"ECS_TASK_DEFINITION": "custom",
				},
				Processes: []*twelvefactor.Process{
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
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
					},
					{
						Type:     "vacuum",
						Image:    image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command:  []string{"./bin/vacuum"},
						Schedule: twelvefactor.CRONSchedule("* * * * *"),
						Quantity: 1,
						Labels: map[string]string{
							"empire.app.process": "vacuum",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Nproc:     256,
					},
				},
			},
		},

		{
			newTemplate,
			"task-role.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					"ECS_TASK_DEFINITION":    "custom",
					"EMPIRE_X_TASK_ROLE_ARN": "arn:aws:iam::897883143566:role/stage/app/r101-pg-loadtest",
				},
				Processes: []*twelvefactor.Process{
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
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
					},
					{
						Type:     "vacuum",
						Image:    image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command:  []string{"./bin/vacuum"},
						Schedule: twelvefactor.CRONSchedule("* * * * *"),
						Quantity: 1,
						Labels: map[string]string{
							"empire.app.process": "vacuum",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Nproc:     256,
					},
				},
			},
		},

		{
			newTemplate,
			"cron.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Processes: []*twelvefactor.Process{
					{
						Type:     "send-emails",
						Image:    image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command:  []string{"./bin/send-emails"},
						Schedule: twelvefactor.CRONSchedule("* * * * *"),
						Quantity: 1,
						Labels: map[string]string{
							"empire.app.process": "send-emails",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Nproc:     256,
					},
					{
						Type:     "vacuum",
						Image:    image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command:  []string{"./bin/vacuum"},
						Schedule: twelvefactor.CRONSchedule("* * * * *"),
						Quantity: 0,
						Labels: map[string]string{
							"empire.app.process": "vacuum",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Nproc:     256,
					},
				},
			},
		},

		{
			func() *EmpireTemplate {
				t := newTemplate()
				t.AccessLoggingPolicy = func(app *twelvefactor.Manifest, process *twelvefactor.Process) *AccessLoggingPolicy {
					return &AccessLoggingPolicy{
						Enabled:        aws.Bool(true),
						S3BucketName:   aws.String("accesslogs"),
						S3BucketPrefix: AccessLoggingBucketPrefix(app, process),
					}
				}
				return t
			},
			"access-logging.json",
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Processes: []*twelvefactor.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"PORT": "8080",
						},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
					},
					{
						Type:    "http",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Env: map[string]string{
							"PORT":               "8080",
							"LOAD_BALANCER_TYPE": "alb",
						},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "http",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
					},
				},
			},
		},
	}

	stackTags := []*cloudformation.Tag{
		{Key: aws.String("environment"), Value: aws.String("test")},
	}

	for _, tt := range tests {
		t.Run(tt.file, func(t *testing.T) {
			tmpl := tt.newTemplate()
			tmpl.NoCompress = true
			buf := new(bytes.Buffer)

			filename := fmt.Sprintf("templates/%s", tt.file)
			data := &TemplateData{tt.app, stackTags}
			err := tmpl.Execute(buf, data)
			if assert.NoError(t, err) {
				expected, err := ioutil.ReadFile(filename)
				assert.NoError(t, err)

				if got, want := buf.String(), string(expected); got != want {
					ioutil.WriteFile(filename, buf.Bytes(), 0660)
					t.Errorf("expected generated template to match existing %s. Wrote to %s", tt.file, filename)
				}
			}
		})
	}
}

func TestEmpireTemplate_Errors(t *testing.T) {
	tests := []struct {
		err error
		app *twelvefactor.Manifest
	}{
		{
			// When using an ALB, the container ports must all
			// match.
			errors.New("AWS Application Load Balancers can only map listeners to a single container port. 2 unique container ports were defined: [80 => 80, 8080 => 8080]"),
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					"LOAD_BALANCER_TYPE": "alb",
				},
				Processes: []*twelvefactor.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 80,
									Protocol:  &twelvefactor.HTTP{},
								},
								{
									Host:      8080,
									Container: 8080,
									Protocol:  &twelvefactor.HTTP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Env: map[string]string{
							"PORT": "8080",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
					},
				},
			},
		},

		{
			// When using an ALB, SSL and TCP listeners are not
			// supported.
			errors.New("tcp listeners are not supported with AWS Application Load Balancing"),
			&twelvefactor.Manifest{
				AppID:   "1234",
				Release: "v1",
				Name:    "acme-inc",
				Env: map[string]string{
					"LOAD_BALANCER_TYPE": "alb",
				},
				Processes: []*twelvefactor.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Exposure: &twelvefactor.Exposure{
							Ports: []twelvefactor.Port{
								{
									Host:      80,
									Container: 80,
									Protocol:  &twelvefactor.TCP{},
								},
							},
						},
						Labels: map[string]string{
							"empire.app.process": "web",
						},
						Env: map[string]string{
							"PORT": "8080",
						},
						Memory:    128 * bytesize.MB,
						CPUShares: 256,
						Quantity:  1,
						Nproc:     256,
					},
				},
			},
		},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			tmpl := newTemplate()
			buf := new(bytes.Buffer)
			data := &TemplateData{tt.app, nil}
			err := tmpl.Execute(buf, data)
			assert.Equal(t, tt.err, err)
		})
	}
}

func TestEmpireTemplate_Large(t *testing.T) {
	labels := make(map[string]string)
	env := make(map[string]string)
	for i := 0; i < 100; i++ {
		env[fmt.Sprintf("ENV_VAR_%d", i)] = fmt.Sprintf("value%d", i)
	}
	app := &twelvefactor.Manifest{
		AppID:   "",
		Release: "v1",
		Name:    "bigappwithlotsofprocesses",
		Env:     env,
		Labels:  labels,
	}

	for i := 0; i < 60; i++ {
		app.Processes = append(app.Processes, &twelvefactor.Process{
			Type:    fmt.Sprintf("%d", i),
			Command: []string{"./bin/web"},
		})
	}

	tmpl := newTemplate()
	buf := new(bytes.Buffer)

	data := &TemplateData{app, nil}
	err := tmpl.Execute(buf, data)
	t.Logf("Template size: %d bytes", buf.Len())
	assert.NoError(t, err)
	assert.Condition(t, func() bool {
		return buf.Len() < MaxTemplateSize
	}, fmt.Sprintf("template must be smaller than %d, was %d", MaxTemplateSize, buf.Len()))
}

func TestScheduleExpression(t *testing.T) {
	tests := []struct {
		schedule   twelvefactor.Schedule
		expression string
	}{
		{twelvefactor.CRONSchedule("0 12 * * ? *"), "cron(0 12 * * ? *)"},
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
