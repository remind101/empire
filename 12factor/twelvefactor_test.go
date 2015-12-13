package twelvefactor

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMergeEnv(t *testing.T) {
	tests := []struct {
		in  []map[string]string
		out map[string]string
	}{
		{
			[]map[string]string{
				{"a": "b"},
				{"b": "a"},
			},
			map[string]string{"a": "b", "b": "a"},
		},

		{
			[]map[string]string{
				{"a": "b"},
				{"a": "c"},
			},
			map[string]string{"a": "c"},
		},
	}

	for _, tt := range tests {
		out := MergeEnv(tt.in...)
		assert.Equal(t, out, tt.out)
	}
}
