// Command echosrv is the host-side acceptance peer for the esp-emu network
// probes (_demo/esp-idf-wifi): a TCP echo server on 127.0.0.1:9000. The
// emulator's user-mode NAT (slirp-style) aliases its gateway address
// (192.168.4.1 in the default 192.168.4.0/24 subnet) to the host, and
// forwards each guest connection to the same port on the host's loopback,
// which is where this server listens. The emu firmware dials 192.168.4.1:9000
// and this process echoes the bytes back over the same connection.
//
// Usage:
//
//	go run ./idfproj/host/echosrv [listen-addr]   (default 127.0.0.1:9000)
//
// Every connection and byte count is logged so the firmware-side probe
// output can be cross-checked. Connections idle out after 30 seconds, and
// io.Copy echoes until the peer closes, so the firmware controls the
// connection lifetime.
package main

import (
	"io"
	"log"
	"net"
	"os"
	"time"
)

func main() {
	addr := "127.0.0.1:9000"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		log.Fatalf("echosrv: listen %s: %v", addr, err)
	}
	log.Printf("echosrv: listening on %s", ln.Addr())
	for {
		conn, err := ln.Accept()
		if err != nil {
			log.Fatalf("echosrv: accept: %v", err)
		}
		go echo(conn)
	}
}

// echo copies everything back until the peer closes; each connection is
// bounded by a 30s deadline so a stuck firmware probe cannot leak one.
func echo(conn net.Conn) {
	defer conn.Close()
	log.Printf("echosrv: connection from %s", conn.RemoteAddr())
	if err := conn.SetDeadline(time.Now().Add(30 * time.Second)); err != nil {
		log.Printf("echosrv: set deadline: %v", err)
		return
	}
	n, err := io.Copy(conn, conn)
	log.Printf("echosrv: echoed %d bytes to %s (err=%v)", n, conn.RemoteAddr(), err)
}
