package relay

import (
	"bufio"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

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

	buf := bufio.NewReader(conn)
	str, err := buf.ReadString('\n')
	if err != nil {
		logger.Log(ctx, "at", "ServeTCP", "err", err)
		return
	}
	name := strings.TrimRight(str, "\r\n")

	if c, ok := h.relay.sessions[name]; ok {
		logger.Log(ctx, "at", "HandleConn", "name", name, "container exists, attaching.")

		fmt.Fprintln(conn, "Creating container...")
		if err := h.relay.CreateContainer(ctx, c); err != nil {
			fmt.Fprintln(conn, err.Error())
			logger.Log(ctx, "at", "CreateContainer", "err", err)
			return
		}

		fmt.Fprintln(conn, "Attaching to container...")
		errCh := make(chan error, 2)
		go func() {
			err := h.relay.AttachToContainer(ctx, name, conn, conn)
			if err != nil {
				logger.Log(ctx, "at", "AttachToContainer", "err", err)
			}
			errCh <- err
		}()

		fmt.Fprintln(conn, "Starting container...")
		if err := h.relay.StartContainer(ctx, name); err != nil {
			fmt.Fprintln(conn, err.Error())
			logger.Log(ctx, "at", "StartContainer", "err", err)
			return
		}

		logger.Log(ctx, "at", "WaitContainer", "name", name)
		go func() {
			_, err := h.relay.WaitContainer(ctx, name)
			errCh <- err
		}()

		// Wait for Attach or Wait to finish.
		err := <-errCh
		logger.Log(ctx, "at", "finished-attach-or-wait", "err", err)

		if err := h.relay.PurgeContainer(ctx, name); err != nil {
			logger.Log(ctx, "at", "PurgeContainer", "err", err)
		}
	} else {
		logger.Log(ctx, "at", "HandleConn", "container", name, "container does not exist.")
	}
}
