package cloudformation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/remind101/empire/pkg/bytesize"
	"github.com/remind101/empire/pkg/image"
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
				ID:   "1234",
				Name: "acme-inc",
				Processes: []*scheduler.Process{
					{
						Type:    "web",
						Image:   image.Image{Repository: "remind101/acme-inc", Tag: "latest"},
						Command: []string{"./bin/web"},
						Exposure: &scheduler.Exposure{
							Type: &scheduler.HTTPExposure{},
						},
						FLabels: map[string]string{
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
						FLabels: map[string]string{
							"empire.app.process": "worker",
						},
						FEnv: map[string]string{
							"FOO": "BAR",
						},
					},
				},
			},
		},

		{
			"https.json",
			&scheduler.App{
				ID:   "1234",
				Name: "acme-inc",
				Processes: []*scheduler.Process{
					{
						Type:    "web",
						Command: []string{"./bin/web"},
						Exposure: &scheduler.Exposure{
							Type: &scheduler.HTTPSExposure{
								Cert: "iamcert",
							},
						},
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
		ID:     "",
		Name:   "bigappwithlotsofprocesses",
		Env:    env,
		Labels: labels,
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
	assert.NoError(t, err)
	assert.Condition(t, func() bool {
		return buf.Len() < MaxTemplateSize
	}, fmt.Sprintf("template must be smaller than %d, was %d", MaxTemplateSize, buf.Len()))
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
	}
}
