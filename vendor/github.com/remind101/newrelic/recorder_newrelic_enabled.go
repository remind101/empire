// +build newrelic_enabled

package newrelic

import (
	"runtime"
	"time"

	"github.com/remind101/newrelic/sdk"
)

// recorder is the default implementation of the Recorder interface. It
// records CPU and Memory metrics.
type recorder struct {
	interval time.Duration
}

func newRecorder(interval time.Duration) *recorder {
	return &recorder{interval: interval}
}

func (r *recorder) Interval() time.Duration {
	return r.interval
}

func (r *recorder) Record() error {
	return recordMemory()
}

func recordMemory() error {
	m := &runtime.MemStats{}
	runtime.ReadMemStats(m)
	mb := float64(m.Alloc) / (1024 * 1024)
	_, err := sdk.RecordMemoryUsage(mb)
	return err
}
