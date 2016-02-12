package cloudwatch

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
)

type RejectedLogEventsInfoError struct {
	Info *cloudwatchlogs.RejectedLogEventsInfo
}

func (e *RejectedLogEventsInfoError) Error() string {
	return fmt.Sprintf("log messages were rejected")
}

// Writer is an io.Writer implementation that writes lines to a cloudwatch logs
// stream.
type Writer struct {
	group, stream, sequenceToken *string

	client client

	closed bool
	err    error

	events eventsBuffer

	throttle <-chan time.Time

	sync.Mutex // This protects calls to flush.
}

func NewWriter(group, stream string, client *cloudwatchlogs.CloudWatchLogs) *Writer {
	w := &Writer{
		group:    aws.String(group),
		stream:   aws.String(stream),
		client:   client,
		throttle: time.Tick(writeThrottle),
	}
	go w.start() // start flushing
	return w
}

// Write takes b, and creates cloudwatch log events for each individual line.
// If Flush returns an error, subsequent calls to Write will fail.
func (w *Writer) Write(b []byte) (int, error) {
	if w.closed {
		return 0, io.ErrClosedPipe
	}

	if w.err != nil {
		return 0, w.err
	}

	return w.buffer(b)
}

// starts continously flushing the buffered events.
func (w *Writer) start() error {
	for {
		// Exit if the stream is closed.
		if w.closed {
			return nil
		}

		<-w.throttle
		if err := w.Flush(); err != nil {
			return err
		}
	}
}

// Closes the writer. Any subsequent calls to Write will return
// io.ErrClosedPipe.
func (w *Writer) Close() error {
	w.closed = true
	return w.Flush() // Flush remaining buffer.
}

// Flush flushes the events that are currently buffered.
func (w *Writer) Flush() error {
	w.Lock()
	defer w.Unlock()

	events := w.events.drain()

	// No events to flush.
	if len(events) == 0 {
		return nil
	}

	w.err = w.flush(events)
	return w.err
}

// flush flashes a slice of log events. This method should be called
// sequentially to ensure that the sequence token is updated properly.
func (w *Writer) flush(events []*cloudwatchlogs.InputLogEvent) error {
	resp, err := w.client.PutLogEvents(&cloudwatchlogs.PutLogEventsInput{
		LogEvents:     events,
		LogGroupName:  w.group,
		LogStreamName: w.stream,
		SequenceToken: w.sequenceToken,
	})
	if err != nil {
		return err
	}

	if resp.RejectedLogEventsInfo != nil {
		w.err = &RejectedLogEventsInfoError{Info: resp.RejectedLogEventsInfo}
		return w.err
	}

	w.sequenceToken = resp.NextSequenceToken

	return nil
}

// buffer splits up b into individual log events and inserts them into the
// buffer.
func (w *Writer) buffer(b []byte) (int, error) {
	r := bufio.NewReader(bytes.NewReader(b))

	var (
		n   int
		eof bool
	)

	for !eof {
		b, err := r.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				eof = true
			} else {
				break
			}
		}

		if len(b) == 0 {
			continue
		}

		w.events.add(&cloudwatchlogs.InputLogEvent{
			Message:   aws.String(string(b)),
			Timestamp: aws.Int64(now().UnixNano() / 1000000),
		})

		n += len(b)
	}

	return n, nil
}

// eventsBuffer represents a buffer of cloudwatch events that are protected by a
// mutex.
type eventsBuffer struct {
	sync.Mutex
	events []*cloudwatchlogs.InputLogEvent
}

func (b *eventsBuffer) add(event *cloudwatchlogs.InputLogEvent) {
	b.Lock()
	defer b.Unlock()

	b.events = append(b.events, event)
}

func (b *eventsBuffer) drain() []*cloudwatchlogs.InputLogEvent {
	b.Lock()
	defer b.Unlock()

	events := b.events[:]
	b.events = nil
	return events
}
