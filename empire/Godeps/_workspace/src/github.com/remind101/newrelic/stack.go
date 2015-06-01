package newrelic

import "sync"

// rootSegment is used as the parentID for root segments.
const rootSegment int64 = 0

type SegmentStack struct {
	sync.Mutex
	s []int64
}

func NewSegmentStack() *SegmentStack {
	return &SegmentStack{s: []int64{}}
}

// Push pushes a segment id onto the segment stack.
func (s *SegmentStack) Push(id int64) {
	s.Lock()
	defer s.Unlock()
	s.s = append(s.s, id)
}

// Pop pops a segment id off of the segment stack. It returns false if the stack is empty.
func (s *SegmentStack) Pop() (int64, bool) {
	s.Lock()
	defer s.Unlock()
	if s.Len() == 0 {
		return rootSegment, false
	}
	id := s.s[s.Len()-1]
	s.s = s.s[:s.Len()-1]
	return id, true
}

// Peek returns id from the top of the stack. It returns rootSegment if the stack is empty.
func (s *SegmentStack) Peek() int64 {
	if s.Len() == 0 {
		return rootSegment
	}
	return s.s[s.Len()-1]
}

// Len returns the length of the stack.
func (s *SegmentStack) Len() int {
	return len(s.s)
}
