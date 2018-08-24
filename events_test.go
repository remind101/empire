package empire

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMultiEventStream(t *testing.T) {
	boom := errors.New("boom")
	s := MultiEventStream{EventStreamFunc(func(event Event) error {
		return boom
	})}
	err := s.PublishEvent(RunEvent{App: "acme-inc", Command: []string{"bash"}})
	assert.EqualError(t, err, "1 error(s) occurred:\n\n* boom")

}

func TestEvents_String(t *testing.T) {
	tests := []struct {
		event Event
		out   string
	}{
		// RunEvent
		{RunEvent{App: "acme-inc", Command: []string{"bash"}}, "Started running `bash` (detached) on acme-inc"},
		{RunEvent{App: "acme-inc", Command: []string{"bash"}, Finished: true}, "Ran `bash` (detached) on acme-inc"},
		{RunEvent{App: "acme-inc", Attached: true, Command: []string{"bash"}}, "Started running `bash` (attached) on acme-inc"},
		{RunEvent{App: "acme-inc", URL: "https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#logEvent:group=runs;stream=dac6eaff-6e0b-4708-9277-9f38aea2f528", Attached: true, Command: []string{"bash"}}, "Started running `bash` (attached) on acme-inc (<https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#logEvent:group=runs;stream=dac6eaff-6e0b-4708-9277-9f38aea2f528|logs>)"},

		// RestartEvent
		{RestartEvent{App: "acme-inc"}, "Restarted acme-inc"},
		{RestartEvent{App: "acme-inc", PID: "abcd"}, "Restarted `abcd` on acme-inc"},

		// MaintenanceEvent
		{MaintenanceEvent{App: "acme-inc", Maintenance: false}, "Disabled maintenance mode on acme-inc"},
		{MaintenanceEvent{App: "acme-inc", Maintenance: true}, "Enabled maintenance mode on acme-inc"},

		// ScaleEvent
		{ScaleEvent{
			App: "acme-inc",
			Updates: []*ScaleEventUpdate{
				&ScaleEventUpdate{Process: "web", Quantity: 10, Constraints: Constraints{CPUShare: 1024, Memory: 1024}, PreviousQuantity: 5, PreviousConstraints: Constraints{CPUShare: 1024, Memory: 1024}},
			},
		}, "Scaled `web` on acme-inc from 5(1024:1.00kb) to 10(1024:1.00kb)"},
		{ScaleEvent{
			App: "acme-inc",
			Updates: []*ScaleEventUpdate{
				&ScaleEventUpdate{Process: "web", Quantity: 5, Constraints: Constraints{CPUShare: 1024, Memory: 1024}, PreviousQuantity: 10, PreviousConstraints: Constraints{CPUShare: 1024, Memory: 1024}},
			},
		}, "Scaled `web` on acme-inc from 10(1024:1.00kb) to 5(1024:1.00kb)"},
		{ScaleEvent{
			App: "acme-inc",
			Updates: []*ScaleEventUpdate{
				&ScaleEventUpdate{Process: "web", Quantity: 5, Constraints: Constraints{CPUShare: 1024, Memory: 1024}, PreviousQuantity: 5, PreviousConstraints: Constraints{CPUShare: 512, Memory: 1024}},
			},
		}, "Scaled `web` on acme-inc from 5(512:1.00kb) to 5(1024:1.00kb)"},
		{ScaleEvent{
			App: "acme-inc",
			Updates: []*ScaleEventUpdate{
				&ScaleEventUpdate{Process: "web", Quantity: 10, PreviousQuantity: 5, PreviousConstraints: Constraints{CPUShare: 512, Memory: 1024}},
			},
		}, "Scaled `web` on acme-inc from 5(512:1.00kb) to 10(512:1.00kb)"},

		// DeployEvent
		{DeployEvent{App: "acme-inc", Image: "remind101/acme-inc:master"}, "Deployed remind101/acme-inc:master to acme-inc"},
		{DeployEvent{Image: "remind101/acme-inc:master"}, "Deployed remind101/acme-inc:master"},

		// RollbackEvent
		{RollbackEvent{App: "acme-inc", Version: 1}, "Rolled back acme-inc to v1"},

		// SetEvent
		{SetEvent{App: "acme-inc", Changed: []string{"RAILS_ENV"}}, "Changed environment variables on acme-inc (RAILS_ENV)"},

		// CreateEvent
		{CreateEvent{Name: "acme-inc"}, "Created acme-inc"},

		// DestroyEvent
		{DestroyEvent{App: "acme-inc"}, "Destroyed acme-inc"},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("%d", i), func(t *testing.T) {
			out := tt.event.String()
			assert.Equal(t, tt.out, out)
		})
	}
}
