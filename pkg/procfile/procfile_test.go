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
		strings.NewReader(`web: ./bin/web`),
		Procfile{
			"web": ProcessDefinition{
				Command: "./bin/web",
			},
		},
	},

	// Extended Procfile with health checks and http exposure.
	{
		strings.NewReader(`web:
  command: ./bin/web
  health_checks:
    - type: http
      path: /health
      timeout: 10
      interval: 30`),
		Procfile{
			"web": ProcessDefinition{
				Command: "./bin/web",
				HealthChecks: []HealthCheck{
					HTTPHealthCheck{
						Path:     "/health",
						Timeout:  10,
						Interval: 30,
					},
				},
			},
		},
	},

	// Extended Procfile with health checks and http exposure.
	{
		strings.NewReader(`web:
  command: ./bin/web
  health_checks:
    - type: tcp`),
		Procfile{
			"web": ProcessDefinition{
				Command: "./bin/web",
				HealthChecks: []HealthCheck{
					TCPHealthCheck{},
				},
			},
		},
	},
}

func TestParse(t *testing.T) {
	for _, tt := range parseTests {
		p, err := Parse(tt.in)
		assert.NoError(t, err)
		assert.Equal(t, tt.out, p)
	}
}
