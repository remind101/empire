package stats

import (
	"runtime"
	"time"
)

// Runtime enters into a loop, sampling and outputing the runtime stats periodically.
func Runtime(stats Stats) {
	SampleEvery(stats, 30*time.Second)
}

// SampleEvery enters a loop, sampling at the specified interval
func SampleEvery(stats Stats, t time.Duration) {
	c := time.Tick(t)
	for _ = range c {
		r := newRuntimeSample()
		r.drain(stats)
	}
}

// runtimeSample represents a sampling of the runtime stats.
type runtimeSample struct {
	*runtime.MemStats
	NumGoroutine int
}

// newRuntimeSample samples the current runtime and returns a RuntimeSample.
func newRuntimeSample() *runtimeSample {
	r := &runtimeSample{MemStats: &runtime.MemStats{}}
	runtime.ReadMemStats(r.MemStats)
	r.NumGoroutine = runtime.NumGoroutine()
	return r
}

// drain drains all of the metrics.
func (r *runtimeSample) drain(stats Stats) {
	stats.Gauge("runtime.NumGoroutine", float32(r.NumGoroutine), 1.0, nil)
	stats.Gauge("runtime.MemStats.Alloc", float32(r.MemStats.Alloc), 1.0, nil)
	stats.Gauge("runtime.MemStats.Frees", float32(r.MemStats.Frees), 1.0, nil)
	stats.Gauge("runtime.MemStats.HeapAlloc", float32(r.MemStats.HeapAlloc), 1.0, nil)
	stats.Gauge("runtime.MemStats.HeapIdle", float32(r.MemStats.HeapIdle), 1.0, nil)
	stats.Gauge("runtime.MemStats.HeapObjects", float32(r.MemStats.HeapObjects), 1.0, nil)
	stats.Gauge("runtime.MemStats.HeapReleased", float32(r.MemStats.HeapReleased), 1.0, nil)
	stats.Gauge("runtime.MemStats.HeapSys", float32(r.MemStats.HeapSys), 1.0, nil)
	stats.Gauge("runtime.MemStats.LastGC", float32(r.MemStats.LastGC), 1.0, nil)
	stats.Gauge("runtime.MemStats.Lookups", float32(r.MemStats.Lookups), 1.0, nil)
	stats.Gauge("runtime.MemStats.Mallocs", float32(r.MemStats.Mallocs), 1.0, nil)
	stats.Gauge("runtime.MemStats.MCacheInuse", float32(r.MemStats.MCacheInuse), 1.0, nil)
	stats.Gauge("runtime.MemStats.MCacheSys", float32(r.MemStats.MCacheSys), 1.0, nil)
	stats.Gauge("runtime.MemStats.MSpanInuse", float32(r.MemStats.MSpanInuse), 1.0, nil)
	stats.Gauge("runtime.MemStats.MSpanSys", float32(r.MemStats.MSpanSys), 1.0, nil)
	stats.Gauge("runtime.MemStats.NextGC", float32(r.MemStats.NextGC), 1.0, nil)
	stats.Gauge("runtime.MemStats.NumGC", float32(r.MemStats.NumGC), 1.0, nil)
	stats.Gauge("runtime.MemStats.StackInuse", float32(r.MemStats.StackInuse), 1.0, nil)
}
