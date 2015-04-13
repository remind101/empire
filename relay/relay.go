package relay

import (
	"bufio"
	"fmt"
	"net"

	"code.google.com/p/go-uuid/uuid"

	"github.com/remind101/pkg/logger"
	"golang.org/x/net/context"
)

var (
	// DefaultOptions is a default Options instance that can be passed when
	// intializing a new Relay.
	DefaultOptions = Options{}
)

type Options struct{}

type Relay struct {
	sessions map[string]bool
}

// New returns a new Relay instance.
func New(options Options) *Relay {
	return &Relay{
		sessions: map[string]bool{},
	}
}

func (r *Relay) HandleConn(ctx context.Context, conn net.Conn) {
	defer conn.Close()
	logger.Log(ctx, "at", "HandleConn", "received new tcp connection.")

	scanner := bufio.NewScanner(conn)
	scanner.Scan()
	if err := scanner.Err(); err != nil {
		fmt.Printf("reading standard input:", err)
	}
	session := scanner.Text()
	if ok := r.sessions[session]; ok {
		logger.Log(ctx, "at", "HandleConn", "session", session, "session exists.")
	} else {
		logger.Log(ctx, "at", "HandleConn", "session", session, "session does not exist.")
	}

	// w := io.MultiWriter(os.Stdout, conn)
	// go io.Copy(w, conn)
	// go io.Copy(conn, os.Stdin)
}

func (r *Relay) NewSession() string {
	id := uuid.New()
	r.sessions[id] = true
	return id
}
