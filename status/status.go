package status

import "sync"

// Status represents a status update from an action within Empire.
type Status interface {
	// Returns a human readable string about the status update.
	String() string
}

// status implements the Status interface
type status struct {
	message string
}

func (s *status) String() string {
	return s.message
}

// NewStatus returns the default implementation of the Status interface
func NewStatus(m string) *status {
	return &status{message: m}
}

// StatusStream is an interface for publishing status updates while executing
// an Empire action.
type StatusStream interface {
	// Publish publishes an update to the status stream
	Publish(Status)

	// Done finalizes the status stream
	Done(error)
}

type SubscribableStream interface {
	Subscribe() <-chan Status
	Error() error
}

// stream implements the StatusStream interface with support for subscribing to
// updates published to the stream.
type stream struct {
	sync.Mutex
	done bool
	err  error
	ch   chan Status
}

// NewStatusStream returns a new instance of the default status stream.
func NewStatusStream() *stream {
	return &stream{ch: make(chan Status, 100)}
}

func (s *stream) Publish(status Status) {
	s.Lock()
	defer s.Unlock()

	if s.done {
		// TODO look into using log here
		panic("Publish called on finalized status stream")
	}

	s.publish(status)
}

func (s *stream) publish(status Status) {
	select {
	case s.ch <- status:
	default:
		// Drop
	}
}

func (s *stream) Subscribe() <-chan Status {
	return s.ch
}

func (s *stream) Done(err error) {
	s.Lock()
	defer s.Unlock()

	if !s.done {
		s.done = true
		s.err = err
		close(s.ch)
	}
}

func (s *stream) Error() error {
	return s.err
}
