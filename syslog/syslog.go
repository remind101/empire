package syslog

import (
	"log"
	"net"
)

const BUFFER_SIZE = 65507 // Practical limit on UDP packet size

type Handler interface {
	Handle([]byte)
}

type Server struct {
	handlers []Handler
	conn     *net.UDPConn
}

func NewServer() *Server {
	return &Server{
		handlers: make([]Handler, 0),
	}
}

func (s *Server) Start(p string) {
	if p == "" {
		p = "10514"
	}

	addr, err := net.ResolveUDPAddr("udp", "0.0.0.0:"+p)
	if err != nil {
		panic(err)
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		panic(err)
	}

	log.Println("Listening on", addr.String())

	s.conn = conn
}

func (s *Server) Serve() {
	buf := make([]byte, BUFFER_SIZE)

	for {
		n, _, err := s.conn.ReadFromUDP(buf)
		if err != nil {
			log.Println("error=", err)
		}

		for _, h := range s.handlers {
			// Create a new byte slice for each handler
			b := make([]byte, n)
			copy(b, buf[0:n])
			h.Handle(b)
		}
	}

}

func (s *Server) AddHandler(h Handler) {
	s.handlers = append(s.handlers, h)
}

func (s *Server) Close() {
	s.conn.Close()
}
