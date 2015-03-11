package main

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/heroku/log-shuttle"
	"github.com/remind101/empire/syslog"
)

func main() {
	shuttleUrls := os.Getenv("SHUTTLE_URLS")
	if shuttleUrls == "" {
		log.Printf("Usage: SHUTTLE_URLS=https://logs1.com,https://logs2.com %s\n", os.Args[0])
		os.Exit(1)
	}
	urls := strings.Split(shuttleUrls, ",")

	sh := &shuttleHandler{}
	sh.Launch(urls)

	// Start UDP server
	server := syslog.NewServer()
	server.AddHandler(sh)

	server.Start(os.Getenv("PORT"))
	defer server.Close()

	server.Serve()
}

type shuttleHandler struct {
	shuttles []*shuttle.Shuttle
}

func (s *shuttleHandler) Launch(urls []string) {
	s.shuttles = make([]*shuttle.Shuttle, len(urls))

	for i, u := range urls {
		config := shuttle.NewConfig()
		config.InputFormat = shuttle.InputFormatRFC5424
		config.LogsURL = u
		s.shuttles[i] = shuttle.NewShuttle(config)
		s.shuttles[i].Launch()
		log.Println("Launching shuttle for", u)
	}
}

func (s *shuttleHandler) Handle(b []byte) {
	ll := shuttle.NewLogLine(b, time.Now())

	// Enqueue in all shuttles
	for _, sh := range s.shuttles {
		sh.Enqueue(ll)
	}

}
