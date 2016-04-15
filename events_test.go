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
		{RunEvent{User: "ejholmes", App: "acme-inc", Command: []string{"bash"}}, "ejholmes ran `bash` (detached) on acme-inc"},
		{RunEvent{User: "ejholmes", App: "acme-inc", Attached: true, Command: []string{"bash"}}, "ejholmes ran `bash` (attached) on acme-inc"},
		{RunEvent{User: "ejholmes", App: "acme-inc", URL: "https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#logEvent:group=runs;stream=dac6eaff-6e0b-4708-9277-9f38aea2f528", Attached: true, Command: []string{"bash"}}, "ejholmes ran `bash` (attached) on acme-inc (<https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#logEvent:group=runs;stream=dac6eaff-6e0b-4708-9277-9f38aea2f528|logs>)"},

		// RestartEvent
		{RestartEvent{User: "ejholmes", App: "acme-inc"}, "ejholmes restarted acme-inc"},
		{RestartEvent{User: "ejholmes", App: "acme-inc", PID: "abcd"}, "ejholmes restarted `abcd` on acme-inc"},

		// ScaleEvent
		{ScaleEvent{User: "ejholmes", App: "acme-inc", Process: "web", Quantity: 10, Constraints: Constraints{CPUShare: 1024, Memory: 1024}, PreviousQuantity: 5, PreviousConstraints: Constraints{CPUShare: 1024, Memory: 1024}}, "ejholmes scaled `web` on acme-inc from 5(1024:1.00kb) to 10(1024:1.00kb)"},
		{ScaleEvent{User: "ejholmes", App: "acme-inc", Process: "web", Quantity: 5, Constraints: Constraints{CPUShare: 1024, Memory: 1024}, PreviousQuantity: 10, PreviousConstraints: Constraints{CPUShare: 1024, Memory: 1024}}, "ejholmes scaled `web` on acme-inc from 10(1024:1.00kb) to 5(1024:1.00kb)"},
		{ScaleEvent{User: "ejholmes", App: "acme-inc", Process: "web", Quantity: 5, Constraints: Constraints{CPUShare: 1024, Memory: 1024}, PreviousQuantity: 5, PreviousConstraints: Constraints{CPUShare: 512, Memory: 1024}}, "ejholmes scaled `web` on acme-inc from 5(512:1.00kb) to 5(1024:1.00kb)"},
		{ScaleEvent{User: "ejholmes", App: "acme-inc", Process: "web", Quantity: 10, PreviousQuantity: 5, PreviousConstraints: Constraints{CPUShare: 512, Memory: 1024}}, "ejholmes scaled `web` on acme-inc from 5(512:1.00kb) to 10(512:1.00kb)"},

		// DeployEvent
		{DeployEvent{User: "ejholmes", App: "acme-inc", Image: "remind101/acme-inc:master", Environment: "production", Release: 32}, "ejholmes deployed remind101/acme-inc:master to acme-inc production (v32)"},
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
