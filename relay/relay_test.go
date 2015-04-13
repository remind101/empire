package relay

import (
	"crypto/tls"
	"fmt"
	"net"

	"github.com/remind101/empire/relay/tcp"
)

// TODO replace with tcp.Server
type TestTCPServer struct {
	Addr     string
	Listener net.Listener
	TLS      *tls.Config
	Config   *tcp.Server
}

func NewTestTCPServer() (*TCPServer, error) {
	go relay.ServeTCP(l)
	return nil
}

func NewUnstartedTCPServer() *TCPServer {
	return &TCPServer{
		Listener: newLocalListener(),
	}
}

func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("httptest: failed to listen on a port: %v", err))
		}
	}
	return l
}

func (s *TCPServer) Start() {
	if s.Addr != "" {
		panic("TCP Server already started")
	}
	s.Addr = s.Listener.Addr().String()
	go s.Config.Serve(s.Listener)
}
