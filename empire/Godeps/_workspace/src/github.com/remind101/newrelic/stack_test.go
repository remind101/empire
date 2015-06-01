package newrelic

import "testing"

func TestStack(t *testing.T) {
	s := NewSegmentStack()

	if got, want := s.Peek(), rootSegment; got != want {
		t.Errorf("s.Peek() => %s; want %s", got, want)
	}

	s.Push(1)
	if got, want := s.Peek(), int64(1); got != want {
		t.Errorf("s.Peek() => %s; want %s", got, want)
	}

	s.Push(2)
	if got, want := s.Peek(), int64(2); got != want {
		t.Errorf("s.Peek() => %s; want %s", got, want)
	}

	i, _ := s.Pop()
	if got, want := i, int64(2); got != want {
		t.Errorf("s.Pop() => %s; want %s", got, want)
	}

	i, _ = s.Pop()
	if got, want := i, int64(1); got != want {
		t.Errorf("s.Pop() => %s; want %s", got, want)
	}

	i, _ = s.Pop()
	if got, want := i, int64(0); got != want {
		t.Errorf("s.Pop() => %s; want %s", got, want)
	}
}
