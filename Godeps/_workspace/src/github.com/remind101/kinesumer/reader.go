package kinesumer

import (
	"io"

	k "github.com/remind101/kinesumer/interface"
)

// Reader provides an io.Reader implementation that can read data from a kinesis
// stream.
type Reader struct {
	records <-chan k.Record

	// buffered data for the current record.
	buf []byte
	// done is called when the current buffer is fully consumed.
	done func()
}

// NewReader returns a new Reader instance that reads data from records.
func NewReader(records <-chan k.Record) *Reader {
	return &Reader{records: records}
}

// Read implements io.Reader Read. Read will copy <= len(b) bytes from the kinesis
// stream into b.
func (r *Reader) Read(b []byte) (n int, err error) {
	for {
		// If there's no data in the buffer, we'll grab the next record
		// and set the internal buffer to point to the data in that
		// record. When all data from buf is read, Done() will be called
		// on the record.
		if len(r.buf) == 0 {
			select {
			case record, ok := <-r.records:
				if !ok {
					// Channel is closed, return io.EOF.
					err = io.EOF
					return
				}

				r.buf = record.Data()
				r.done = record.Done
			default:
				// By convention, Read should return rather than wait
				// for data to become available. If no data is available
				// at this time, we'll return what we've copied
				// so far.
				return
			}
		}

		n += r.copy(b[n:])
		if n == len(b) {
			return
		}
	}

	return
}

// copy copies as much as it can from r.buf into b. If it succeeds in copying
// all of the data, r.done is called.
func (r *Reader) copy(b []byte) (n int) {
	n += copy(b, r.buf)
	if len(r.buf) >= n {
		// If there's still some buffered data left, truncate the buffer
		// and return.
		r.buf = r.buf[n:]
	}

	if len(r.buf) == 0 {
		r.done()
	}

	return
}
