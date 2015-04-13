package relay

import (
	"fmt"
	"log"
	"net"

	"golang.org/x/net/context"
)

func ListenAndServeTCP(ctx context.Context, r *Relay, tcpPort string) {
	log.Printf("Starting tcp server on port %s\n", tcpPort)
	ln, err := net.Listen("tcp", fmt.Sprintf(":%s", tcpPort))
	if err != nil {
		panic(err)
	}
	defer ln.Close()

	for {
		// TODO handle Temporary net errs just like http.Server does
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		go r.HandleConn(ctx, conn)
	}
}
