package main

import (
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/heroku/log-shuttle"
)

const BUFFER_SIZE = 1024 * 1024

func main() {
	shuttleUrls := os.Getenv("SHUTTLE_URLS")
	if shuttleUrls == "" {
		log.Printf("Usage: SHUTTLE_URLS=https://logs1.com,https://logs2.com %s\n", os.Args[0])
		os.Exit(1)
	}
	urls := strings.Split(shuttleUrls, ",")

	// Launch the shuttles!
	shuttles := make([]*shuttle.Shuttle, len(urls))
	for i, u := range urls {
		config := shuttle.NewConfig()
		config.InputFormat = shuttle.InputFormatRFC5424
		config.LogsURL = u
		shuttles[i] = shuttle.NewShuttle(config)
		shuttles[i].Launch()
		log.Println("Launching shuttle for", u)
	}

	// Start UDP server
	server := &server{}
	server.Start(os.Getenv("PORT"))
	defer server.Close()

	// Pipe syslog messages into the shuttles
	buf := make([]byte, BUFFER_SIZE)

	for {
		n, _, err := server.conn.ReadFromUDP(buf)

		// Create a log line
		b := make([]byte, n)
		copy(b, buf[0:n])
		ll := shuttle.NewLogLine(b, time.Now())

		// Enqueue in all shuttles
		for _, s := range shuttles {
			s.Enqueue(ll)
		}

		if err != nil {
			log.Println("Error: ", err)
		}
	}
}

type server struct {
	conn *net.UDPConn
}

func (s *server) Start(p string) {
	if p == "" {
		p = "10514"
	}

	addr, err := net.ResolveUDPAddr("udp", ":"+p)
	check(err)

	conn, err := net.ListenUDP("udp", addr)
	check(err)

	log.Println("Listening on", addr.String())

	s.conn = conn
}

func (s *server) Close() {
	s.conn.Close()
}

// check panics when given an error.
func check(err error) {
	if err != nil {
		panic(err)
	}
}
