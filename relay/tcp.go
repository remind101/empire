package relay

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"

	"github.com/remind101/empire/relay/tcp"
	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

func NewTCPHandler(r *Relay) tcp.Handler {
	return &commonTCPHandler{
		handler: &containerSession{relay: r},
	}
}

type commonTCPHandler struct {
	handler tcp.Handler
}

func (h *commonTCPHandler) ServeTCP(ctx context.Context, conn net.Conn) {
	// Add a logger to the context
	l := logger.New(log.New(os.Stdout, "", 0))
	ctx = logger.WithLogger(ctx, l)

	h.handler.ServeTCP(ctx, conn)
}

type containerSession struct {
	relay *Relay
}

func (h *containerSession) ServeTCP(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	logger.Log(ctx, "at", "HandleConn", "received new tcp connection.")

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		fmt.Printf("reading standard input:", err)
	}
	session := scanner.Text()
	if ok := h.relay.sessions[session]; ok {
		logger.Log(ctx, "at", "HandleConn", "session", session, "session exists.")
		fmt.Fprintf(conn, "Connection accepted for session %s\r\n", session)
	} else {
		logger.Log(ctx, "at", "HandleConn", "session", session, "session does not exist.")
	}

	// w := io.MultiWriter(os.Stdout, conn)
	// go io.Copy(w, conn)
	// go io.Copy(conn, os.Stdin)
}
