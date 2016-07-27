package stats

import (
	"time"

	"github.com/cactus/go-statsd-client/statsd"
)

// Statsd is an implementation of the Stats interface backed by statsd.
type Statsd struct {
	client statsd.Statter
}

// NewStatsd returns a new Statsd implementation that sends stats to addr.
func NewStatsd(addr, prefix string) (*Statsd, error) {
	c, err := statsd.NewClient(addr, prefix)
	if err != nil {
		return nil, err
	}
	return &Statsd{
		client: c,
	}, nil
}

func (s *Statsd) Inc(name string, value int64, rate float32, tags []string) error {
	return s.client.Inc(name, value, rate)
}

func (s *Statsd) Timing(name string, value time.Duration, rate float32, tags []string) error {
	return s.client.TimingDuration(name, value, rate)
}

func (s *Statsd) Gauge(name string, value float32, rate float32, tags []string) error {
	return s.client.Gauge(name, int64(value), rate)
}
