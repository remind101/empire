package stats

import (
	"time"

	"github.com/DataDog/datadog-go/statsd"
)

// Dogstatsd provides an implementation of the Stats interface backed by
// dogstatsd.
type Dogstatsd struct {
	*statsd.Client
}

// NewDogstatsd returns a new Dogstatsd instance that sends statsd metrics to addr.
func NewDogstatsd(addr string) (*Dogstatsd, error) {
	c, err := statsd.New(addr)
	if err != nil {
		return nil, err
	}

	return &Dogstatsd{
		Client: c,
	}, nil
}

func (s *Dogstatsd) Inc(name string, value int64, rate float32, tags []string) error {
	return s.Client.Count(name, value, tags, float64(rate))
}

func (s *Dogstatsd) Timing(name string, value time.Duration, rate float32, tags []string) error {
	timeInMilliseconds := float64(value / time.Millisecond)
	return s.Client.TimeInMilliseconds(name, timeInMilliseconds, tags, float64(rate))
}

func (s *Dogstatsd) Gauge(name string, value float32, rate float32, tags []string) error {
	return s.Client.Gauge(name, float64(value), tags, float64(rate))
}

func (s *Dogstatsd) Histogram(name string, value float32, rate float32, tags []string) error {
	return s.Client.Histogram(name, float64(value), tags, float64(rate))
}
