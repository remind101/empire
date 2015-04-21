package relay

import (
	"fmt"
	"net/http/httptest"

	"github.com/remind101/empire/relay/tcptest"
)

func NewTestHTTPServer(r *Relay) *httptest.Server {
	if r == nil {
		r = NewTestRelay()
	}
	return httptest.NewServer(NewHTTPHandler(r))
}

func NewTestTCPServer(r *Relay) *tcptest.Server {
	if r == nil {
		r = NewTestRelay()
	}
	return tcptest.NewServer(NewTCPHandler(r))
}

func NewTestRelay() *Relay {
	o := Options{}

	o.Docker.Socket = "fake"
	o.Tcp.Host = "rendezvous://rendez.empire.example.com"
	o.Tcp.Port = "5000"

	sid := 0
	o.ContainerNameFunc = func(s string) string {
		sid++
		return fmt.Sprintf("%s.%d", s, sid)
	}
	return New(o)
}
