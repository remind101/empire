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

	// Extended Procfile with health checks and http exposure.
	{
		strings.NewReader(`---
web:
  command:
    - nginx
    - -g
    - daemon off;`),
		ExtendedProcfile{
			"web": Process{
				Command: []interface{}{
					"nginx",
					"-g",
					"daemon off;",
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
