// Package stats provides an interface for instrumenting Empire.
package stats

import (
	"context"
	"time"
)

// Stats provides an interface for generating instruments, like guages and
// counts.
type Stats interface {
	Inc(name string, value int64, rate float32, tags []string) error
	Timing(name string, value time.Duration, rate float32, tags []string) error
	Gauge(name string, value float32, rate float32, tags []string) error
	Histogram(name string, value float32, rate float32, tags []string) error
}

type nullStats struct{}

func (s *nullStats) Inc(name string, value int64, rate float32, tags []string) error {
	return nil
}

func (s *nullStats) Timing(name string, value time.Duration, rate float32, tags []string) error {
	return nil
}

func (s *nullStats) Gauge(name string, value float32, rate float32, tags []string) error {
	return nil
}

func (s *nullStats) Histogram(name string, value float32, rate float32, tags []string) error {
	return nil
}

var Null = &nullStats{}

// taggedStats wraps a Stats implementation to include some additional tags.
type taggedStats struct {
	tags  []string
	stats Stats
}

func (s *taggedStats) Inc(name string, value int64, rate float32, tags []string) error {
	return s.stats.Inc(name, value, rate, append(tags, s.tags...))
}

func (s *taggedStats) Timing(name string, value time.Duration, rate float32, tags []string) error {
	return s.stats.Timing(name, value, rate, append(tags, s.tags...))
}

func (s *taggedStats) Gauge(name string, value float32, rate float32, tags []string) error {
	return s.stats.Gauge(name, value, rate, append(tags, s.tags...))
}

func (s *taggedStats) Histogram(name string, value float32, rate float32, tags []string) error {
	return s.stats.Histogram(name, value, rate, append(tags, s.tags...))
}

// WithStats returns a new context.Context with the Stats implementation
// embedded.
func WithStats(ctx context.Context, stats Stats) context.Context {
	return context.WithValue(ctx, statsKey, stats)
}

// FromContext returns the Stats implementation that's embedded in the context.
func FromContext(ctx context.Context) (Stats, bool) {
	stats, ok := ctx.Value(statsKey).(Stats)
	return stats, ok
}

// WithTags will return a context.Context where all metrics recorded with the
// embedded Stats implementation will include the given stats.
func WithTags(ctx context.Context, tags []string) context.Context {
	stats, ok := FromContext(ctx)
	if !ok {
		return ctx
	}
	return WithStats(ctx, &taggedStats{tags, stats})
}

func Inc(ctx context.Context, name string, value int64, rate float32, tags []string) error {
	if stats, ok := FromContext(ctx); ok {
		return stats.Inc(name, value, rate, tags)
	}
	return nil
}

func Timing(ctx context.Context, name string, value time.Duration, rate float32, tags []string) error {
	if stats, ok := FromContext(ctx); ok {
		return stats.Timing(name, value, rate, tags)
	}
	return nil
}

func Gauge(ctx context.Context, name string, value float32, rate float32, tags []string) error {
	if stats, ok := FromContext(ctx); ok {
		return stats.Gauge(name, value, rate, tags)
	}
	return nil
}

func Histogram(ctx context.Context, name string, value float32, rate float32, tags []string) error {
	if stats, ok := FromContext(ctx); ok {
		return stats.Histogram(name, value, rate, tags)
	}
	return nil
}

type key int

const (
	statsKey key = iota
)
