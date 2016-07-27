package stats

import (
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

// DataDog provides an implementation of the Stats interface backed by
// dogstatsd.
type DataDog struct {
	*statsd.Client
}

// NewDataDog returns a new DataDog instance that sends statsd metrics to addr.
func NewDataDog(addr string) (*DataDog, error) {
	c, err := statsd.New(addr)
	if err != nil {
		return nil, err
	}

	return &DataDog{
		Client: c,
	}, nil
}

func (s *DataDog) Inc(name string, value int64, rate float32, tags []string) error {
	return s.Client.Count(name, value, tags, float64(rate))
}

func (s *DataDog) Timing(name string, value time.Duration, rate float32, tags []string) error {
	timeInMilliseconds := float64(value / time.Millisecond)
	return s.Client.TimeInMilliseconds(name, timeInMilliseconds, tags, float64(rate))
}
