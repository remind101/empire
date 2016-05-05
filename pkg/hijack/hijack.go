package hijack

import (
	"fmt"
	"io"
	"net/http"
	"sync"
)

// HijackReadWriter is an io.Writer and io.ReadCloser implementation that will
// defer hijacking a connection until a read or write is attempted.
type HijackReadWriter struct {
	sync.Mutex

	Header   string
	Response http.ResponseWriter

	hijacked bool
	// writer is returned when hijacking the connection
	writer io.Writer
	// reader is returned when hijacking the connection
	reader io.ReadCloser
}

func (rw *HijackReadWriter) Write(b []byte) (int, error) {
	if err := rw.hijack(); err != nil {
		return 0, err
	}
	return rw.writer.Write(b)
}

func (rw *HijackReadWriter) Read(b []byte) (int, error) {
	if err := rw.hijack(); err != nil {
		return 0, err
	}
	return rw.reader.Read(b)
}

func (rw *HijackReadWriter) Close() {
	closeStreams(rw.reader, rw.writer)
}

func (rw *HijackReadWriter) hijack() error {
	rw.Lock()
	defer rw.Unlock()

	if !rw.hijacked {
		reader, writer, err := hijackServer(rw.Response)
		if err != nil {
			return err
		}
		rw.reader = reader
		rw.writer = writer
		fmt.Fprintf(writer, rw.Header)
		rw.hijacked = true
	}
	return nil
}

func hijackServer(w http.ResponseWriter) (io.ReadCloser, io.Writer, error) {
	conn, _, err := w.(http.Hijacker).Hijack()
	if err != nil {
		return nil, nil, err
	}
	// Flush the options to make sure the client sets the raw mode
	conn.Write([]byte{})
	return conn, conn, nil
}

func closeStreams(streams ...interface{}) {
	for _, stream := range streams {
		if tcpc, ok := stream.(interface {
			CloseWrite() error
		}); ok {
			tcpc.CloseWrite()
		} else if closer, ok := stream.(io.Closer); ok {
			closer.Close()
		}
	}
}
