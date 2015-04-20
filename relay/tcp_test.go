package relay

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"testing"
)

func TestHandshakeSessionNotFound(t *testing.T) {
	ts := NewTestTCPServer(nil)
	defer ts.Close()

	conn, err := net.Dial("tcp", ts.Addr)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Fprintf(conn, "non-existing-session-id\r\n")
	_, err = bufio.NewReader(conn).ReadString('\n')
	if err != io.EOF {
		t.Fatal("Expected TCP Server to close connection with no response.")
	}
}

func TestHandshakeValidSession(t *testing.T) {
	r := NewTestRelay()
	r.sessions["1"] = &Container{}

	ts := NewTestTCPServer(r)
	defer ts.Close()

	conn, err := net.Dial("tcp", ts.Addr)
	if err != nil {
		t.Fatal(err)
	}

	fmt.Fprintf(conn, "1\r\n") // The session generated above

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	if got, want := scanner.Text(), "Creating container..."; got != want {
		t.Errorf("Response from TCP Server => %q; want %q", got, want)
	}
}
