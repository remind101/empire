package customresources

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntValue(t *testing.T) {
	type foo struct {
		I IntValue `json:"I"`
	}

	tests := []struct {
		in  []byte
		out foo
	}{
		{[]byte(`{"I": 1}`), foo{I: 1}},
		{[]byte(`{"I": "1"}`), foo{I: 1}},
	}

	for _, tt := range tests {
		var i foo
		err := json.Unmarshal(tt.in, &i)
		assert.NoError(t, err)
		assert.Equal(t, tt.out, i)
	}
}
