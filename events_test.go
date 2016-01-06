package empire

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvents_String(t *testing.T) {
	tests := []struct {
		event Event
		out   string
	}{
		// RunEvent
		{RunEvent{User: "ejholmes", App: "acme-inc", Command: "bash"}, "ejholmes ran `bash` (detached) on acme-inc"},
		{RunEvent{User: "ejholmes", App: "acme-inc", Attached: true, Command: "bash"}, "ejholmes ran `bash` (attached) on acme-inc"},

		// RestartEvent
		{RestartEvent{User: "ejholmes", App: "acme-inc"}, "ejholmes restarted acme-inc"},
		{RestartEvent{User: "ejholmes", App: "acme-inc", PID: "abcd"}, "ejholmes restarted `abcd` on acme-inc"},

		// ScaleEvent
		{ScaleEvent{User: "ejholmes", App: "acme-inc", Process: "web", Quantity: 10, PreviousQuantity: 5}, "ejholmes scaled `web` on acme-inc from 5 to 10 (+5)"},
		{ScaleEvent{User: "ejholmes", App: "acme-inc", Process: "web", Quantity: 5, PreviousQuantity: 10}, "ejholmes scaled `web` on acme-inc from 10 to 5 (-5)"},

		// DeployEvent
		{DeployEvent{User: "ejholmes", App: "acme-inc", Image: "remind101/acme-inc:master"}, "ejholmes deployed remind101/acme-inc:master to acme-inc"},
		{DeployEvent{User: "ejholmes", Image: "remind101/acme-inc:master"}, "ejholmes deployed remind101/acme-inc:master"},

		// RollbackEvent
		{RollbackEvent{User: "ejholmes", App: "acme-inc", Version: 1}, "ejholmes rolled back acme-inc to v1"},

		// SetEvent
		{SetEvent{User: "ejholmes", App: "acme-inc", Changed: []string{"RAILS_ENV"}}, "ejholmes changed environment variables on acme-inc (RAILS_ENV)"},

		// CreateEvent
		{CreateEvent{User: "ejholmes", Name: "acme-inc"}, "ejholmes created acme-inc"},

		// DestroyEvent
		{DestroyEvent{User: "ejholmes", App: "acme-inc"}, "ejholmes destroyed acme-inc"},
	}

	for _, tt := range tests {
		out := tt.event.String()
		assert.Equal(t, tt.out, out)
	}
}
