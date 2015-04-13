package tcptest

import (
	"crypto/tls"
	"fmt"
	"net"
	"sync"

	"github.com/remind101/empire/relay/tcp"
)

type Server struct {
	Addr     string
	Listener net.Listener
	TLS      *tls.Config
	Config   *tcp.Server
}

func NewServer(h tcp.Handler) *Server {
	s := NewUnstartedTCPServer(h)
	s.Start()
	return s
}

func NewUnstartedTCPServer(h tcp.Handler) *Server {
	return &Server{
		Listener: newLocalListener(),
		Config:   &tcp.Server{Handler: h},
	}
}

func (s *Server) Start() {
	if s.Addr != "" {
		panic("TCP Server already started")
	}
	s.Listener = &historyListener{Listener: s.Listener}
	s.Addr = s.Listener.Addr().String()
	go s.Config.Serve(s.Listener)
}

func (s *Server) Close() {
	s.Listener.Close()
	s.CloseClientConnections()
}

func (s *Server) CloseClientConnections() {
	hl, ok := s.Listener.(*historyListener)
	if !ok {
		return
	}
	hl.Lock()
	for _, conn := range hl.history {
		conn.Close()
	}
	hl.Unlock()
}

// historyListener keeps track of all connections that it's ever
// accepted.
type historyListener struct {
	net.Listener
	sync.Mutex // protects history
	history    []net.Conn
}

func (hs *historyListener) Accept() (c net.Conn, err error) {
	c, err = hs.Listener.Accept()
	if err == nil {
		hs.Lock()
		hs.history = append(hs.history, c)
		hs.Unlock()
	}
	return
}

func newLocalListener() net.Listener {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		if l, err = net.Listen("tcp6", "[::1]:0"); err != nil {
			panic(fmt.Sprintf("tcptest: failed to listen on a port: %v", err))
		}
	}
	return l
}
