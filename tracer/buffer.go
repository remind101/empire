package tracer

import (
	"math/rand"
	"sync"
)

const (
	spanBufferDefaultMaxSize = 10000
)

// spansBuffer is a threadsafe buffer for spans.
type spansBuffer struct {
	lock    sync.Mutex
	spans   []*Span
	maxSize int
}

func newSpansBuffer(maxSize int) *spansBuffer {

	// small sanity check on the max size.
	if maxSize <= 0 {
		maxSize = spanBufferDefaultMaxSize
	}

	return &spansBuffer{maxSize: maxSize}
}

func (sb *spansBuffer) Push(span *Span) {
	sb.lock.Lock()
	if len(sb.spans) < sb.maxSize {
		sb.spans = append(sb.spans, span)
	} else {
		idx := rand.Intn(sb.maxSize)
		sb.spans[idx] = span
	}
	sb.lock.Unlock()
}

func (sb *spansBuffer) Pop() []*Span {
	sb.lock.Lock()
	defer sb.lock.Unlock()

	if len(sb.spans) == 0 {
		return nil
	}

	// FIXME[matt] on rotation, we could re-use the slices and spans here
	// and avoid re-allocing.
	spans := sb.spans
	sb.spans = nil

	return spans
}

func (sb *spansBuffer) Len() int {
	sb.lock.Lock()
	defer sb.lock.Unlock()
	return len(sb.spans)
}
