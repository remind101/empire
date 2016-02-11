package empire

import "fmt"

// MaintenanceMode is an error implementation that can be returned to indicate
// that Empire is in maintenance mode.
type MaintenanceMode struct {
	// A human readable description for why Empire is in maintenance mode.
	Reason string
}

// Error implements the error interface.
func (e *MaintenanceMode) Error() string {
	return fmt.Sprintf("maintenance mode: %s", e.Reason)
}
