package relay

import (
	"bufio"
	"fmt"
	"net"
)

func ListenTCP() {
	ln, err := net.Listen("tcp", ":9000")
	if err != nil {
		panic(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			panic(err)
		}

		go handleConn(conn)
	}
}

func handleConn(conn net.Conn) {
	buf := bufio.NewReader(conn)
	for {
		line, err := buf.ReadBytes('\n')
		if err != nil {
			fmt.Printf("Error reading bytes: %v", err)
		}
		conn.Write(line)
	}
	// w := io.MultiWriter(os.Stdout, conn)
	// go io.Copy(w, conn)
	// go io.Copy(conn, os.Stdin)
}
