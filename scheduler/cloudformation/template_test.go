package cloudformation

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/route53"
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
						Command: []string{"./bin/web"},
						Exposure: &scheduler.Exposure{
							Type: &scheduler.HTTPExposure{},
						},
					},
					{
						Type:    "worker",
						Command: []string{"./bin/worker"},
						Env: map[string]string{
							"FOO": "BAR",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		tmpl := &EmpireTemplate{
			HostedZone: &route53.HostedZone{
				Id:   aws.String("Z3DG6IL3SJCGPX"),
				Name: aws.String("empire"),
			},
		}
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
