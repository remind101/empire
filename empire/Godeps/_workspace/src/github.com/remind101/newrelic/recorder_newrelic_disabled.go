// +build !newrelic_enabled

package newrelic

import "time"

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
	return nil
}
