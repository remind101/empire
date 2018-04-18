package empire

import (
	"fmt"
	"io"

	"github.com/remind101/empire/pkg/jsonmessage"
)

// DeploymentStream provides a wrapper around an io.Writer for writing
// jsonmessage statuses, and implements the scheduler.StatusStream interface.
type DeploymentStream struct {
	*jsonmessage.Stream
}

// NewDeploymentStream wraps the io.Writer as a DeploymentStream.
func NewDeploymentStream(w io.Writer) *DeploymentStream {
	return &DeploymentStream{jsonmessage.NewStream(w)}
}

// Status writes a simple status update to the jsonmessage stream.
func (w *DeploymentStream) Status(message string) error {
	m := jsonmessage.JSONMessage{Status: fmt.Sprintf("Status: %s", message)}
	return w.Encode(m)
}

// Error writes the error to the jsonmessage stream. The error that is provided
// is also returned, so that Error() can be used in return values.
func (w *DeploymentStream) Error(err error) error {
	if encErr := w.Encode(jsonmessage.NewError(err)); encErr != nil {
		return encErr
	}
	return err
}
