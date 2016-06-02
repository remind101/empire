package cloudformation

import (
	"encoding/json"
	"testing"

	"github.com/remind101/empire"
	"github.com/stretchr/testify/assert"
)

func TestVarsFromRequest(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"ResourceProperties": {"Environment": {"FOO": "bar", "BAR": null}}}`), &req)
	assert.NoError(t, err)

	bar := "bar"
	vars := varsFromRequest(req)
	assert.Equal(t, empire.Vars{
		"FOO": &bar,
		"BAR": nil,
	}, vars)
}

func TestVarsFromRequest_DeletedVars(t *testing.T) {
	var req Request
	err := json.Unmarshal([]byte(`{"RequestType": "Update", "ResourceProperties": {"Environment": {"FOO": "bar", "BAR": null}}, "OldResourceProperties": {"Environment": {"FOOBAR": "foobar"}}}`), &req)
	assert.NoError(t, err)

	bar := "bar"
	vars := varsFromRequest(req)
	assert.Equal(t, empire.Vars{
		"FOO":    &bar,
		"BAR":    nil,
		"FOOBAR": nil,
	}, vars)
}
