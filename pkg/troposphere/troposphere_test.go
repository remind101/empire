package troposphere

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTemplate(t *testing.T) {
	tests := []struct {
		in  *Template
		out string
	}{
		{
			buildTemplate(func(t *Template) {
				t.Parameters["Cluster"] = Parameter{
					Type:    "String",
					Default: "",
				}
			}),
			`{
  "Conditions": {},
  "Outputs": {},
  "Parameters": {
    "Cluster": {
      "Type": "String",
      "Default": ""
    }
  },
  "Resources": {}
}`,
		},
	}

	for _, tt := range tests {
		raw, err := json.MarshalIndent(tt.in, "", "  ")
		assert.NoError(t, err)
		assert.Equal(t, tt.out, string(raw))
	}
}

func buildTemplate(f func(*Template)) *Template {
	t := NewTemplate()
	f(t)
	return t
}
