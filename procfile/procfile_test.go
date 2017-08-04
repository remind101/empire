package procfile

import (
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ecs"
	"github.com/stretchr/testify/assert"
)

var parseTests = []struct {
	in  io.Reader
	out Procfile
}{
	// Simple standard Procfile.
	{
		strings.NewReader(`---
web: ./bin/web`),
		StandardProcfile{
			"web": "./bin/web",
		},
	},

	// Extended Procfile with health checks and http exposure.
	{
		strings.NewReader(`---
web:
  command: ./bin/web`),
		ExtendedProcfile{
			"web": Process{
				Command: "./bin/web",
			},
		},
	},

	// Extended Procfile with ports.
	{
		strings.NewReader(`---
web:
  command:
    - nginx
    - -g
    - daemon off;
  ports:
    - "80:8080"
    - "443"
    - 52`),
		ExtendedProcfile{
			"web": Process{
				Command: []interface{}{
					"nginx",
					"-g",
					"daemon off;",
				},
				Ports: []Port{
					{
						Host:      80,
						Container: 8080,
					},
					{
						Host:      443,
						Container: 443,
					},
					{
						Host:      52,
						Container: 52,
					},
				},
			},
		},
	},

	// Ports with protocol
	{
		strings.NewReader(`---
web:
  command:
    - nginx
    - -g
    - daemon off;
  ports:
  - "80:8080":
      protocol: "tcp"`),
		ExtendedProcfile{
			"web": Process{
				Command: []interface{}{
					"nginx",
					"-g",
					"daemon off;",
				},
				Ports: []Port{
					{
						Host:      80,
						Container: 8080,
						Protocol:  "tcp",
					},
				},
			},
		},
	},

	// Environment variables.
	{
		strings.NewReader(`---
web:
  command:
    - nginx
    - -g
    - daemon off;
  environment:
    ENABLE_FOO: "true"`),
		ExtendedProcfile{
			"web": Process{
				Command: []interface{}{
					"nginx",
					"-g",
					"daemon off;",
				},
				Environment: map[string]string{
					"ENABLE_FOO": "true",
				},
			},
		},
	},

	// ECS placement constraints
	{
		strings.NewReader(`---
web:
  command: nginx
  ecs:
    placement_constraints:
      - type: memberOf
        expression: "attribute:ecs.instance-type =~ t2.*"
    placement_strategy:
      - type: spread
        field: "attribute:ecs.availability-zone"`),
		ExtendedProcfile{
			"web": Process{
				Command: "nginx",
				ECS: &ECS{
					PlacementConstraints: []*ecs.PlacementConstraint{
						{Type: aws.String("memberOf"), Expression: aws.String("attribute:ecs.instance-type =~ t2.*")},
					},
					PlacementStrategy: []*ecs.PlacementStrategy{
						{Type: aws.String("spread"), Field: aws.String("attribute:ecs.availability-zone")},
					},
				},
			},
		},
	},
}

func TestParse(t *testing.T) {
	for i, tt := range parseTests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			t.Log(tt.in)
			p, err := Parse(tt.in)
			assert.NoError(t, err)
			assert.Equal(t, tt.out, p)
		})
	}
}
