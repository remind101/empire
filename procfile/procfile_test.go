package procfile

import (
	"io"
	"strings"
	"testing"

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
}

func TestParse(t *testing.T) {
	for _, tt := range parseTests {
		t.Log(tt.in)
		p, err := Parse(tt.in)
		assert.NoError(t, err)
		assert.Equal(t, tt.out, p)
	}
}
