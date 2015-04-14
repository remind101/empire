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
	o := DefaultOptions
	o.Host = "rendezvous://rendez.empire.example.com"
	sid := 0
	o.SessionGenerator = func() string {
		sid++
		return fmt.Sprintf("%d", sid)
	}
	return New(o)
}
