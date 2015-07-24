// Package http provides streaming implementations of various net/http types.
package http

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ResponseWriter provides an implementation of the http.ResponseWriter
// interface that is unbuffered. Every call to write will be flushed to the
// underyling connection and streamed to the client.
type ResponseWriter struct {
	http.ResponseWriter
	w io.Writer
}

// StreamingResponseWriter wraps the http.ResponseWriter with unbuffered
// streaming. If the provided ResponseWriter does not implement http.Flusher,
// this function will panic.
func StreamingResponseWriter(w http.ResponseWriter) *ResponseWriter {
	fw, err := newFlushWriter(w)
	if err != nil {
		panic(err)
	}

	return &ResponseWriter{
		ResponseWriter: w,
		w:              fw,
	}
}

// Write delegates to the underlying flushWriter to perform the write and flush
// it to the connection.
func (w *ResponseWriter) Write(p []byte) (int, error) {
	return w.w.Write(p)
}

// flushWriter is an io.Writer implementation that flushes to the underlying
// io.Writer whenever Write is called.
type flushWriter struct {
	f http.Flusher
	w io.Writer
}

func newFlushWriter(w http.ResponseWriter) (*flushWriter, error) {
	fw := &flushWriter{w: w}
	if f, ok := w.(http.Flusher); ok {
		fw.f = f
	} else {
		return fw, errors.New("provided http.ResponseWriter does not implement http.Flusher")
	}
	return fw, nil
}

func (fw *flushWriter) Write(p []byte) (n int, err error) {
	n, err = fw.w.Write(p)
	if fw.f != nil {
		fw.f.Flush()
	}
	return
}

// Heartbeat sends the null character periodically, to keep the connection alive.
func Heartbeat(outStream io.Writer, interval time.Duration) chan struct{} {
	stop := make(chan struct{})
	t := time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-t.C:
				fmt.Fprintf(outStream, "\x00")
				continue
			case <-stop:
				t.Stop()
				return
			}
		}
	}()

	return stop
}
