package empire

import (
	"fmt"
	"io"

	"golang.org/x/net/context"
)

const (
	// GenericProcessName is the process name for `emp run` processes not defined in the procfile.
	GenericProcessName = "run"
)

// RunRecorder is a function that returns an io.Writer that will be written to
// to record Stdout and Stdin of interactive runs.
type RunRecorder func() (io.Writer, error)

type runnerService struct {
	*Empire
}

func (r *runnerService) Run(ctx context.Context, opts RunOpts) error {
	return fmt.Errorf("`emp run` is currently unsupported")
}
